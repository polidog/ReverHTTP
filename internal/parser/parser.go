package parser

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/polidog/reverhttp/internal/ast"
	"github.com/polidog/reverhttp/internal/lexer"
	"github.com/polidog/reverhttp/internal/token"
)

// Parser is a recursive descent parser for ReverHTTP DSL.
type Parser struct {
	l      *lexer.Lexer
	cur    token.Token
	peek   token.Token
	errors []string
}

// New creates a new Parser.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	// Read two tokens to fill cur and peek.
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns the list of parse errors.
func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(msg string) {
	pos := p.cur.Pos
	p.errors = append(p.errors, fmt.Sprintf("%s:%d:%d: %s", pos.File, pos.Line, pos.Column, msg))
}

func (p *Parser) nextToken() {
	p.cur = p.peek
	p.peek = p.l.NextToken()
}

func (p *Parser) curIs(t token.Type) bool {
	return p.cur.Type == t
}

func (p *Parser) peekIs(t token.Type) bool {
	return p.peek.Type == t
}

func (p *Parser) expect(t token.Type) bool {
	if p.peekIs(t) {
		p.nextToken()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s (%q)", t, p.peek.Type, p.peek.Literal))
	return false
}

func (p *Parser) skipNewlines() {
	for p.curIs(token.NEWLINE) {
		p.nextToken()
	}
}

// skipToNextStatement skips tokens until a recovery point is found.
func (p *Parser) skipToNextStatement() {
	for !p.curIs(token.EOF) {
		if p.curIs(token.PIPE) || p.curIs(token.NEWLINE) || token.IsHTTPMethod(p.cur.Type) {
			return
		}
		p.nextToken()
	}
}

// ParseFile parses a complete .rever file.
func (p *Parser) ParseFile() *ast.File {
	file := &ast.File{}

	p.skipNewlines()

	for !p.curIs(token.EOF) {
		switch {
		case p.curIs(token.IMPORT):
			imp := p.parseImport()
			if imp != nil {
				file.Imports = append(file.Imports, imp)
			}
		case p.curIs(token.TYPE):
			td := p.parseType()
			if td != nil {
				file.Types = append(file.Types, td)
			}
		case p.curIs(token.DEFAULTS):
			file.Defaults = p.parseDefaults()
		case token.IsHTTPMethod(p.cur.Type):
			route := p.parseRoute()
			if route != nil {
				file.Routes = append(file.Routes, route)
			}
		default:
			p.addError(fmt.Sprintf("unexpected token %s (%q)", p.cur.Type, p.cur.Literal))
			p.nextToken()
		}
		p.skipNewlines()
	}

	return file
}

// parseImport parses:
//
//	import <alias> = <source>@<version>
//	import <alias> = @/<path>
func (p *Parser) parseImport() *ast.ImportDecl {
	pos := p.cur.Pos
	p.nextToken() // skip 'import'

	if !p.curIs(token.IDENT) && !p.curIs(token.DELETE) {
		p.addError(fmt.Sprintf("expected identifier after 'import', got %s", p.cur.Type))
		p.skipToNextStatement()
		return nil
	}

	alias := p.cur.Literal
	p.nextToken()

	if !p.curIs(token.ASSIGN) {
		p.addError("expected '=' after import alias")
		p.skipToNextStatement()
		return nil
	}
	p.nextToken() // skip '='

	decl := &ast.ImportDecl{Pos: pos, Alias: alias}

	// Check for local import: @/path
	if p.curIs(token.AT) && p.peekIs(token.SLASH) {
		decl.Local = true
		p.nextToken() // skip '@'
		p.nextToken() // skip '/'
		// Build the path: everything until newline or EOF
		var pathParts []string
		for !p.curIs(token.NEWLINE) && !p.curIs(token.EOF) {
			pathParts = append(pathParts, p.cur.Literal)
			p.nextToken()
		}
		decl.Source = "@/" + strings.Join(pathParts, "")
		return decl
	}

	// Remote import: github.com/reverhttp/std-fetch@0.1.0
	source := p.parseImportSource()
	decl.Source = source

	// Check for @version
	if p.curIs(token.AT) {
		p.nextToken() // skip '@'
		// Version: read tokens and join directly (e.g., "0" "." "1" "." "0" → "0.1.0")
		var verParts []string
		for !p.curIs(token.NEWLINE) && !p.curIs(token.EOF) {
			verParts = append(verParts, p.cur.Literal)
			p.nextToken()
		}
		decl.Version = strings.Join(verParts, "")
	}

	return decl
}

func (p *Parser) parseImportSource() string {
	var parts []string
	for !p.curIs(token.AT) && !p.curIs(token.NEWLINE) && !p.curIs(token.EOF) {
		parts = append(parts, p.cur.Literal)
		p.nextToken()
	}
	// Reconstruct: tokens like "github", ".", "com", "/", "reverhttp", "/", "std-fetch"
	// The lexer produces IDENT, DOT, IDENT, SLASH, IDENT, SLASH, IDENT
	return joinSourceParts(parts)
}

func joinSourceParts(parts []string) string {
	return strings.Join(parts, "")
}

// parseType parses:
//
//	type User { id: int, name: string }
func (p *Parser) parseType() *ast.TypeDecl {
	pos := p.cur.Pos
	p.nextToken() // skip 'type'

	if !p.curIs(token.IDENT) {
		p.addError("expected type name after 'type'")
		p.skipToNextStatement()
		return nil
	}

	name := p.cur.Literal
	p.nextToken()

	if !p.curIs(token.LBRACE) {
		p.addError("expected '{' after type name")
		p.skipToNextStatement()
		return nil
	}
	p.nextToken() // skip '{'
	p.skipNewlines()

	td := &ast.TypeDecl{Pos: pos, Name: name}

	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		p.skipNewlines()
		if p.curIs(token.RBRACE) {
			break
		}

		fieldName := p.cur.Literal
		p.nextToken()

		if !p.curIs(token.COLON) {
			p.addError("expected ':' after field name")
			p.skipToNextStatement()
			continue
		}
		p.nextToken() // skip ':'

		typeName := p.cur.Literal
		p.nextToken()

		td.Fields = append(td.Fields, &ast.Field{Name: fieldName, TypeName: typeName})

		// Skip optional comma or newline
		if p.curIs(token.COMMA) {
			p.nextToken()
		}
		p.skipNewlines()
	}

	if p.curIs(token.RBRACE) {
		p.nextToken() // skip '}'
	}

	return td
}

// parseDefaults parses:
//
//	defaults
//	  cors(...)
//	  auth(...)
func (p *Parser) parseDefaults() *ast.DefaultsBlock {
	pos := p.cur.Pos
	p.nextToken() // skip 'defaults'
	p.skipNewlines()

	block := &ast.DefaultsBlock{Pos: pos}

	for p.curIs(token.CACHE) || p.curIs(token.CORS) || p.curIs(token.AUTH) {
		d := p.parseDirective()
		if d != nil {
			block.Directives = append(block.Directives, d)
		}
		p.skipNewlines()
	}

	return block
}

// parseDirective parses a route-level directive: cache(...), cors(...), auth(...)
func (p *Parser) parseDirective() *ast.Directive {
	pos := p.cur.Pos
	name := p.cur.Literal
	p.nextToken() // skip directive name

	d := &ast.Directive{Pos: pos, Name: name}

	if !p.curIs(token.LPAREN) {
		return d
	}
	p.nextToken() // skip '('

	d.Args = p.parseDirectiveArgs()

	if p.curIs(token.RPAREN) {
		p.nextToken() // skip ')'
	}

	// Check for "as <name>" (used by auth)
	if p.curIs(token.AS) {
		p.nextToken() // skip 'as'
		if p.curIs(token.IDENT) {
			d.Bind = p.cur.Literal
			p.nextToken()
		}
	}

	return d
}

func (p *Parser) parseDirectiveArgs() []*ast.Arg {
	var args []*ast.Arg

	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		arg := &ast.Arg{}

		// Check if this is a named arg or keyword
		if p.curIs(token.NONE) {
			// none keyword (e.g., cors(none), auth(none))
			arg.Name = "none"
			arg.Value = ast.Expr{Kind: ast.ExprBool, StrVal: "true"}
			args = append(args, arg)
			p.nextToken()
			if p.curIs(token.COMMA) {
				p.nextToken()
			}
			continue
		}

		if (p.curIs(token.IDENT) || p.isHyphenatedKeyword()) && p.peekIs(token.COLON) {
			// Named arg: name: value
			arg.Name = p.cur.Literal
			p.nextToken() // skip name
			p.nextToken() // skip ':'

			arg.Value = p.parseExprValue()
			args = append(args, arg)
		} else if p.curIs(token.IDENT) || p.curIs(token.INT) || p.curIs(token.STRING) {
			// Positional arg: could be a keyword flag like "public", "private", "bearer"
			// or a value like "credentials"
			arg.Value = p.parseExprValue()
			args = append(args, arg)
		} else {
			p.nextToken()
		}

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	return args
}

func (p *Parser) isHyphenatedKeyword() bool {
	// Check for hyphenated identifiers like "max-age", "expose-headers"
	return p.curIs(token.IDENT)
}

func (p *Parser) parseExprValue() ast.Expr {
	switch {
	case p.curIs(token.STRING):
		val := p.cur.Literal
		p.nextToken()
		return ast.Expr{Kind: ast.ExprString, StrVal: val}
	case p.curIs(token.INT):
		val := p.cur.Literal
		p.nextToken()
		return ast.Expr{Kind: ast.ExprInt, IntVal: val}
	case p.curIs(token.LBRACKET):
		return p.parseListExpr()
	case p.curIs(token.IDENT):
		name := p.cur.Literal
		p.nextToken()
		// Check for function call like hash(user)
		if p.curIs(token.LPAREN) {
			p.nextToken() // skip '('
			argVal := ""
			if p.curIs(token.IDENT) {
				argVal = p.cur.Literal
				p.nextToken()
			}
			if p.curIs(token.RPAREN) {
				p.nextToken() // skip ')'
			}
			return ast.Expr{Kind: ast.ExprFuncCall, StrVal: name + "(" + argVal + ")"}
		}
		// Check for dotted expression
		if p.curIs(token.DOT) {
			parts := []string{name}
			for p.curIs(token.DOT) {
				p.nextToken() // skip '.'
				if p.curIs(token.IDENT) {
					parts = append(parts, p.cur.Literal)
					p.nextToken()
				}
			}
			return ast.Expr{Kind: ast.ExprIdent, StrVal: strings.Join(parts, ".")}
		}
		return ast.Expr{Kind: ast.ExprIdent, StrVal: name}
	default:
		p.nextToken()
		return ast.Expr{}
	}
}

func (p *Parser) parseListExpr() ast.Expr {
	p.nextToken() // skip '['
	var items []string
	for !p.curIs(token.RBRACKET) && !p.curIs(token.EOF) {
		if p.curIs(token.STRING) {
			items = append(items, p.cur.Literal)
		} else {
			items = append(items, p.cur.Literal)
		}
		p.nextToken()
		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}
	if p.curIs(token.RBRACKET) {
		p.nextToken()
	}
	return ast.Expr{Kind: ast.ExprList, ListVal: items}
}

// parseRoute parses a route definition.
func (p *Parser) parseRoute() *ast.Route {
	pos := p.cur.Pos
	method := p.cur.Literal
	p.nextToken() // skip HTTP method

	// Parse path: /users/{id}
	path := p.parsePath()

	route := &ast.Route{Pos: pos, Method: method, Path: path}

	p.skipNewlines()

	// Parse optional directives before first |>
	for p.curIs(token.CACHE) || p.curIs(token.CORS) || p.curIs(token.AUTH) {
		d := p.parseDirective()
		if d != nil {
			route.Directives = append(route.Directives, d)
		}
		p.skipNewlines()
	}

	// Parse pipeline steps
	for p.curIs(token.PIPE) {
		step := p.parsePipelineStep()
		if step != nil {
			route.Steps = append(route.Steps, step)
		}
		p.skipNewlines()
	}

	return route
}

func (p *Parser) parsePath() string {
	var parts []string
	for !p.curIs(token.NEWLINE) && !p.curIs(token.EOF) {
		parts = append(parts, p.cur.Literal)
		p.nextToken()
	}
	return strings.Join(parts, "")
}

func (p *Parser) parsePipelineStep() *ast.PipelineStep {
	pos := p.cur.Pos
	p.nextToken() // skip '|>'

	step := &ast.PipelineStep{Pos: pos}

	switch {
	case p.curIs(token.INPUT):
		step.Kind = ast.StepInput
		step.Input = p.parseInput()
	case p.curIs(token.VALIDATE):
		step.Kind = ast.StepValidate
		step.Validate = p.parseValidate()
	case p.curIs(token.TRANSFORM):
		step.Kind = ast.StepTransform
		step.Transform = p.parseTransform()
	case p.curIs(token.GUARD):
		step.Kind = ast.StepGuard
		step.Guard = p.parseGuard()
	case p.curIs(token.MATCH):
		step.Kind = ast.StepMatch
		step.Match = p.parseMatch()
	case p.curIs(token.RESPOND):
		step.Kind = ast.StepRespond
		step.Respond = p.parseRespond()
	case p.curIs(token.IDENT):
		// Package call: fetch(User, id)
		step.Kind = ast.StepPkgCall
		step.PkgCall = p.parsePkgCall()
	default:
		p.addError(fmt.Sprintf("expected step keyword, got %s (%q)", p.cur.Type, p.cur.Literal))
		p.skipToNextStatement()
		return nil
	}

	// Check for "as <name>"
	if p.curIs(token.AS) {
		p.nextToken() // skip 'as'
		if p.curIs(token.IDENT) {
			step.Bind = p.cur.Literal
			p.nextToken()
		}
	}

	// Check for error flow: ~> status { body }
	if p.curIs(token.ERROR) {
		step.ErrorFlow = p.parseErrorFlow()
	}

	return step
}

// parseInput parses input(id: path.id, name: body.name)
func (p *Parser) parseInput() *ast.InputStep {
	p.nextToken() // skip 'input'
	if !p.curIs(token.LPAREN) {
		p.addError("expected '(' after 'input'")
		return &ast.InputStep{}
	}
	p.nextToken() // skip '('

	input := &ast.InputStep{}

	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		field := &ast.InputField{}

		if p.curIs(token.IDENT) {
			field.Name = p.cur.Literal
			p.nextToken()
		}

		if p.curIs(token.COLON) {
			p.nextToken() // skip ':'
			field.From = p.parseDottedName()
		}

		input.Fields = append(input.Fields, field)

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curIs(token.RPAREN) {
		p.nextToken() // skip ')'
	}

	return input
}

// parseValidate parses validate(id: int & min(1), name: string & min(1) & max(100))
func (p *Parser) parseValidate() *ast.ValidateStep {
	p.nextToken() // skip 'validate'
	if !p.curIs(token.LPAREN) {
		p.addError("expected '(' after 'validate'")
		return &ast.ValidateStep{}
	}
	p.nextToken() // skip '('

	v := &ast.ValidateStep{}

	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		rule := &ast.ValidateRule{}

		if p.curIs(token.IDENT) {
			rule.Field = p.cur.Literal
			p.nextToken()
		}

		if p.curIs(token.COLON) {
			p.nextToken() // skip ':'
			rule.Constraints = p.parseConstraints()
		}

		v.Rules = append(v.Rules, rule)

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curIs(token.RPAREN) {
		p.nextToken() // skip ')'
	}

	return v
}

func (p *Parser) parseConstraints() []*ast.Constraint {
	var constraints []*ast.Constraint

	c := p.parseSingleConstraint()
	if c != nil {
		constraints = append(constraints, c)
	}

	for p.curIs(token.AMPERSAND) {
		p.nextToken() // skip '&'
		c = p.parseSingleConstraint()
		if c != nil {
			constraints = append(constraints, c)
		}
	}

	return constraints
}

func (p *Parser) parseSingleConstraint() *ast.Constraint {
	if !p.curIs(token.IDENT) {
		return nil
	}

	c := &ast.Constraint{Name: p.cur.Literal}
	p.nextToken()

	// Check for args: min(1), max(100), format(email)
	if p.curIs(token.LPAREN) {
		p.nextToken() // skip '('
		for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
			arg := ast.Expr{}
			switch {
			case p.curIs(token.INT):
				arg = ast.Expr{Kind: ast.ExprInt, IntVal: p.cur.Literal}
				p.nextToken()
			case p.curIs(token.STRING):
				arg = ast.Expr{Kind: ast.ExprString, StrVal: p.cur.Literal}
				p.nextToken()
			case p.curIs(token.IDENT):
				arg = ast.Expr{Kind: ast.ExprIdent, StrVal: p.cur.Literal}
				p.nextToken()
			default:
				p.nextToken()
			}
			c.Args = append(c.Args, arg)
			if p.curIs(token.COMMA) {
				p.nextToken()
			}
		}
		if p.curIs(token.RPAREN) {
			p.nextToken()
		}
	}

	return c
}

// parseTransform parses transform(id: int(id), name: trim(name))
func (p *Parser) parseTransform() *ast.TransformStep {
	p.nextToken() // skip 'transform'
	if !p.curIs(token.LPAREN) {
		p.addError("expected '(' after 'transform'")
		return &ast.TransformStep{}
	}
	p.nextToken() // skip '('

	t := &ast.TransformStep{}

	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		field := &ast.TransformField{}

		if p.curIs(token.IDENT) {
			field.Name = p.cur.Literal
			p.nextToken()
		}

		if p.curIs(token.COLON) {
			p.nextToken() // skip ':'
			// Parse function call: int(id), trim(name), lower(email)
			if p.curIs(token.IDENT) {
				field.Func = p.cur.Literal
				p.nextToken()
				if p.curIs(token.LPAREN) {
					p.nextToken() // skip '('
					if p.curIs(token.IDENT) {
						field.From = p.cur.Literal
						p.nextToken()
					}
					if p.curIs(token.RPAREN) {
						p.nextToken() // skip ')'
					}
				}
			}
		}

		t.Fields = append(t.Fields, field)

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curIs(token.RPAREN) {
		p.nextToken() // skip ')'
	}

	return t
}

// parseGuard parses guard <expr> or guard !<expr>
func (p *Parser) parseGuard() *ast.GuardStep {
	p.nextToken() // skip 'guard'

	g := &ast.GuardStep{}

	if p.curIs(token.BANG) {
		g.Negated = true
		p.nextToken() // skip '!'
	}

	if p.curIs(token.IDENT) {
		parts := []string{p.cur.Literal}
		p.nextToken()
		for p.curIs(token.DOT) {
			p.nextToken() // skip '.'
			if p.curIs(token.IDENT) {
				parts = append(parts, p.cur.Literal)
				p.nextToken()
			}
		}
		g.Expr = strings.Join(parts, ".")
	}

	return g
}

// parseMatch parses match <expr> { arms... }
func (p *Parser) parseMatch() *ast.MatchStep {
	p.nextToken() // skip 'match'

	m := &ast.MatchStep{}

	if p.curIs(token.IDENT) {
		m.On = p.parseDottedName()
	}

	if !p.curIs(token.LBRACE) {
		p.addError("expected '{' after match expression")
		return m
	}
	p.nextToken() // skip '{'
	p.skipNewlines()

	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		arm := p.parseMatchArm()
		if arm != nil {
			m.Arms = append(m.Arms, arm)
		}
		p.skipNewlines()
	}

	if p.curIs(token.RBRACE) {
		p.nextToken() // skip '}'
	}

	return m
}

func (p *Parser) parseMatchArm() *ast.MatchArm {
	arm := &ast.MatchArm{}

	// Parse pattern
	if p.curIs(token.UNDERSCORE) {
		arm.IsDefault = true
		arm.Pattern = ast.Pattern{Kind: ast.PatternWildcard, IsDefault: true}
		p.nextToken() // skip '_'
	} else {
		arm.Pattern = p.parsePattern()
	}

	if !p.curIs(token.COLON) {
		p.addError("expected ':' after match pattern")
		p.skipToNextStatement()
		return nil
	}
	p.nextToken() // skip ':'

	// After colon: could be a step, a variable reference, ~> error, or empty (just whitespace then ~>)
	if p.curIs(token.ERROR) {
		arm.ErrorOnly = true
		arm.ErrorFlow = p.parseErrorFlow()
		return arm
	}

	if p.curIs(token.IDENT) {
		name := p.cur.Literal
		// Check if it's a package call (has parentheses after)
		if p.peekIs(token.LPAREN) {
			arm.Step = p.parsePkgCall()
		} else {
			// Just a variable reference like "cached"
			arm.VarRef = name
			p.nextToken()
		}
	}

	// Check for per-arm error flow
	if p.curIs(token.ERROR) {
		arm.ErrorFlow = p.parseErrorFlow()
	}

	return arm
}

func (p *Parser) parsePattern() ast.Pattern {
	// Enable regex mode for pattern parsing
	p.l.SetRegexMode(true)
	defer p.l.SetRegexMode(false)

	pat := ast.Pattern{}

	switch {
	case p.curIs(token.REGEX):
		pat.Kind = ast.PatternRegex
		pat.Regex = p.cur.Literal
		p.nextToken()
		return pat

	case p.curIs(token.STRING):
		first := p.cur.Literal
		p.nextToken()

		// Check for multi-value: "user", "member"
		if p.curIs(token.COMMA) {
			values := []string{first}
			for p.curIs(token.COMMA) {
				p.nextToken() // skip ','
				if p.curIs(token.STRING) {
					values = append(values, p.cur.Literal)
					p.nextToken()
				}
			}
			pat.Kind = ast.PatternMulti
			pat.Values = values
			return pat
		}

		pat.Kind = ast.PatternLiteral
		pat.Value = first
		return pat

	case p.curIs(token.INT):
		first := p.cur.Literal
		p.nextToken()

		// Check for range: 200..299
		if p.curIs(token.RANGE) {
			p.nextToken() // skip '..'
			if p.curIs(token.INT) {
				pat.Kind = ast.PatternRange
				pat.RangeMin = first
				pat.RangeMax = p.cur.Literal
				p.nextToken()
				return pat
			}
		}

		pat.Kind = ast.PatternLiteral
		pat.Value = first
		return pat

	case p.curIs(token.IDENT):
		// Could be bool literal or identifier
		val := p.cur.Literal
		p.nextToken()
		pat.Kind = ast.PatternLiteral
		pat.Value = val
		return pat
	}

	return pat
}

// parsePkgCall parses: fetch(User, id) or redis-cache(key: "user:{id}")
func (p *Parser) parsePkgCall() *ast.PkgCallStep {
	pkg := p.cur.Literal
	p.nextToken() // skip package name

	call := &ast.PkgCallStep{Pkg: pkg}

	if !p.curIs(token.LPAREN) {
		return call
	}
	p.nextToken() // skip '('

	for !p.curIs(token.RPAREN) && !p.curIs(token.EOF) {
		arg := &ast.PkgArg{}

		// Check for named arg: key: "value"
		if p.curIs(token.IDENT) && p.peekIs(token.COLON) {
			// But first check if the identifier is uppercase (type name convention)
			// Named args: the key is lowercase
			if !isUpperCase(p.cur.Literal) {
				arg.Name = p.cur.Literal
				p.nextToken() // skip name
				p.nextToken() // skip ':'
				arg.Value = p.cur.Literal
				p.nextToken()
				call.Args = append(call.Args, arg)
				if p.curIs(token.COMMA) {
					p.nextToken()
				}
				continue
			}
		}

		// Check for object literal: { name, email }
		if p.curIs(token.LBRACE) {
			p.nextToken() // skip '{'
			var fields []string
			for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
				if p.curIs(token.IDENT) {
					fields = append(fields, p.cur.Literal)
					p.nextToken()
				}
				if p.curIs(token.COMMA) {
					p.nextToken()
				}
			}
			if p.curIs(token.RBRACE) {
				p.nextToken()
			}
			arg.ObjectArgs = fields
			call.Args = append(call.Args, arg)
			if p.curIs(token.COMMA) {
				p.nextToken()
			}
			continue
		}

		// Positional arg
		if p.curIs(token.IDENT) {
			arg.Value = p.cur.Literal
			if isUpperCase(p.cur.Literal) {
				arg.IsType = true
			}
			p.nextToken()
		} else if p.curIs(token.INT) {
			arg.Value = p.cur.Literal
			p.nextToken()
		} else if p.curIs(token.STRING) {
			arg.Value = p.cur.Literal
			p.nextToken()
		} else {
			p.nextToken()
		}

		call.Args = append(call.Args, arg)

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curIs(token.RPAREN) {
		p.nextToken() // skip ')'
	}

	return call
}

// parseRespond parses respond <status> [{ body }] [with headers { ... }]
func (p *Parser) parseRespond() *ast.RespondStep {
	p.nextToken() // skip 'respond'

	r := &ast.RespondStep{}

	if p.curIs(token.INT) {
		r.Status = p.cur.Literal
		p.nextToken()
	}

	// Optional body: { key: value, ... }
	if p.curIs(token.LBRACE) {
		r.Body = p.parseBodyFields()
	}

	// Optional: with headers { ... }
	if p.curIs(token.WITH) {
		p.nextToken() // skip 'with'
		if p.curIs(token.HEADERS) {
			p.nextToken() // skip 'headers'
			if p.curIs(token.LBRACE) {
				r.Headers = p.parseBodyFields()
			}
		}
	}

	return r
}

func (p *Parser) parseBodyFields() []*ast.BodyField {
	p.nextToken() // skip '{'
	var fields []*ast.BodyField

	for !p.curIs(token.RBRACE) && !p.curIs(token.EOF) {
		field := &ast.BodyField{}

		if p.curIs(token.IDENT) {
			field.Key = p.cur.Literal
			p.nextToken()
		}

		if p.curIs(token.COLON) {
			p.nextToken() // skip ':'
			field.Value = p.parseFieldValue()
		}

		fields = append(fields, field)

		if p.curIs(token.COMMA) {
			p.nextToken()
		}
	}

	if p.curIs(token.RBRACE) {
		p.nextToken() // skip '}'
	}

	return fields
}

func (p *Parser) parseFieldValue() string {
	if p.curIs(token.STRING) {
		val := p.cur.Literal
		p.nextToken()
		return val
	}

	// Dotted name: user.id, user.name, etc.
	return p.parseDottedName()
}

func (p *Parser) parseDottedName() string {
	var parts []string

	if p.curIs(token.IDENT) {
		parts = append(parts, p.cur.Literal)
		p.nextToken()
	}

	for p.curIs(token.DOT) {
		p.nextToken() // skip '.'
		if p.curIs(token.IDENT) {
			parts = append(parts, p.cur.Literal)
			p.nextToken()
		}
	}

	return strings.Join(parts, ".")
}

// parseErrorFlow parses ~> <status> [{ body }]
func (p *Parser) parseErrorFlow() *ast.ErrorFlow {
	pos := p.cur.Pos
	p.nextToken() // skip '~>'

	ef := &ast.ErrorFlow{Pos: pos}

	if p.curIs(token.INT) {
		ef.Status = p.cur.Literal
		p.nextToken()
	}

	if p.curIs(token.LBRACE) {
		ef.Body = p.parseBodyFields()
	}

	return ef
}

func isUpperCase(s string) bool {
	if len(s) == 0 {
		return false
	}
	return unicode.IsUpper(rune(s[0]))
}
