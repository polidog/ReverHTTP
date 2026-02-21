package ast

import "github.com/polidog/reverhttp/internal/token"

// File is the root AST node representing a .rever file.
type File struct {
	Imports  []*ImportDecl
	Types    []*TypeDecl
	Defaults *DefaultsBlock
	Routes   []*Route
}

// ImportDecl represents an import declaration.
//
//	import fetch = github.com/reverhttp/std-fetch@0.1.0
//	import fetch = @/src/user/fetch.rever
type ImportDecl struct {
	Pos     token.Position
	Alias   string
	Source  string
	Version string // empty for local imports
	Local   bool   // true if source starts with @/
}

// TypeDecl represents a type definition.
//
//	type User { id: int, name: string }
type TypeDecl struct {
	Pos    token.Position
	Name   string
	Fields []*Field
}

// Field represents a field in a type declaration.
type Field struct {
	Name     string
	TypeName string
}

// DefaultsBlock represents a defaults block.
//
//	defaults
//	  cors(...)
//	  auth(...)
type DefaultsBlock struct {
	Pos        token.Position
	Directives []*Directive
}

// Directive represents a route-level directive (cache, cors, auth).
type Directive struct {
	Pos  token.Position
	Name string // "cache", "cors", "auth"
	Args []*Arg
	Bind string // for auth: "as current_user"
}

// Arg is a named or positional argument in a directive or step call.
type Arg struct {
	Name  string // empty for positional args
	Value Expr
}

// Route represents a route definition with its pipeline.
type Route struct {
	Pos        token.Position
	Method     string
	Path       string
	Directives []*Directive
	Steps      []*PipelineStep
}

// PipelineStep represents a step in a pipeline.
type PipelineStep struct {
	Pos       token.Position
	Kind      StepKind
	Input     *InputStep
	Validate  *ValidateStep
	Transform *TransformStep
	Guard     *GuardStep
	Match     *MatchStep
	PkgCall   *PkgCallStep
	Respond   *RespondStep
	Bind      string     // "as name"
	ErrorFlow *ErrorFlow // "~> status { body }"
}

// StepKind indicates which step variant is active.
type StepKind int

const (
	StepInput StepKind = iota
	StepValidate
	StepTransform
	StepGuard
	StepMatch
	StepPkgCall
	StepRespond
)

// InputStep represents input(...).
type InputStep struct {
	Fields []*InputField
}

// InputField represents a field in input().
type InputField struct {
	Name string
	From string // e.g., "path.id", "body.name", "header.x-role"
}

// ValidateStep represents validate(...).
type ValidateStep struct {
	Rules []*ValidateRule
}

// ValidateRule represents a single validation rule.
type ValidateRule struct {
	Field       string
	Constraints []*Constraint
}

// Constraint represents a single validation constraint like int, min(1), max(100), format(email).
type Constraint struct {
	Name string
	Args []Expr
}

// TransformStep represents transform(...).
type TransformStep struct {
	Fields []*TransformField
}

// TransformField represents a field transformation.
type TransformField struct {
	Name string
	Func string // function name: "int", "trim", "lower", etc.
	From string // source variable
}

// GuardStep represents guard <expr>.
type GuardStep struct {
	Negated bool
	Expr    string // the expression (variable name)
}

// MatchStep represents match <expr> { ... }.
type MatchStep struct {
	On   string // the expression to match on
	Arms []*MatchArm
}

// MatchArm represents a single arm of a match expression.
type MatchArm struct {
	Pattern   Pattern
	Step      *PkgCallStep // the step to execute (could also be just a variable ref)
	IsDefault bool
	ErrorFlow *ErrorFlow
	// For default arms that are just an error
	ErrorOnly bool
	// For arms that just reference a variable
	VarRef string
}

// Pattern represents a match pattern.
type Pattern struct {
	Kind      PatternKind
	Value     string   // for literal
	Values    []string // for multi-value
	RangeMin  string   // for range
	RangeMax  string   // for range
	Regex     string   // for regex
	IsDefault bool     // for wildcard _
}

// PatternKind indicates the kind of match pattern.
type PatternKind int

const (
	PatternLiteral PatternKind = iota
	PatternMulti
	PatternRange
	PatternRegex
	PatternWildcard
)

// PkgCallStep represents a call to an imported package step.
//
//	fetch(User, id)
//	create(User, { name, email })
type PkgCallStep struct {
	Pkg  string
	Args []*PkgArg
}

// PkgArg represents an argument to a package call.
type PkgArg struct {
	Name       string // named arg key (e.g., "key" in redis-cache(key: "..."))
	Value      string // simple value
	IsType     bool   // true if this is a type name (starts with uppercase)
	ObjectArgs []string // for { name, email } shorthand
}

// RespondStep represents respond <status> [{ body }] [with headers { ... }].
type RespondStep struct {
	Status  string
	Body    []*BodyField
	Headers []*BodyField
}

// BodyField represents a key-value pair in a respond body or headers.
type BodyField struct {
	Key   string
	Value string // expression like "user.id" or a string literal
}

// ErrorFlow represents ~> <status> [{ body }].
type ErrorFlow struct {
	Pos    token.Position
	Status string
	Body   []*BodyField
}

// Expr is a simple expression — for now, just a string value or an int.
type Expr struct {
	Kind    ExprKind
	StrVal  string
	IntVal  string
	ListVal []string
}

// ExprKind indicates the type of expression.
type ExprKind int

const (
	ExprString ExprKind = iota
	ExprInt
	ExprIdent
	ExprBool
	ExprList
	ExprFuncCall // for things like hash(user)
)

// FuncCallExpr extends Expr for function calls in directive args.
type FuncCallExpr struct {
	Func string
	Arg  string
}
