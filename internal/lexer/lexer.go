package lexer

import (
	"unicode"

	"github.com/polidog/reverhttp/internal/token"
)

// Lexer tokenizes ReverHTTP source code.
type Lexer struct {
	input   string
	file    string
	pos     int  // current position in input
	readPos int  // next read position
	ch      byte // current character
	line    int
	col     int

	// Bracket nesting depth — newlines are suppressed inside brackets.
	parenDepth   int // ()
	braceDepth   int // {}
	bracketDepth int // []

	// regexMode is set by the parser when `/` should be read as regex delimiter.
	regexMode bool
}

// New creates a new Lexer for the given input.
func New(input, file string) *Lexer {
	l := &Lexer{input: input, file: file, line: 1, col: 0}
	l.readChar()
	return l
}

// SetRegexMode enables or disables regex mode. In regex mode, `/` starts a regex literal.
func (l *Lexer) SetRegexMode(on bool) {
	l.regexMode = on
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *Lexer) curPos() token.Position {
	return token.Position{File: l.file, Line: l.line, Column: l.col}
}

func (l *Lexer) newToken(t token.Type, lit string) token.Token {
	return token.Token{Type: t, Literal: lit, Pos: l.curPos()}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespaceAndComments()

	pos := l.curPos()

	switch l.ch {
	case 0:
		return token.Token{Type: token.EOF, Literal: "", Pos: pos}

	case '\n':
		l.line++
		l.col = 0
		l.readChar()
		// Suppress newlines inside brackets
		if l.insideBrackets() {
			return l.NextToken()
		}
		return token.Token{Type: token.NEWLINE, Literal: "\n", Pos: pos}

	case '|':
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.PIPE, Literal: "|>", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.ILLEGAL, Literal: "|", Pos: pos}

	case '~':
		if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.ERROR, Literal: "~>", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.ILLEGAL, Literal: "~", Pos: pos}

	case '&':
		l.readChar()
		return token.Token{Type: token.AMPERSAND, Literal: "&", Pos: pos}

	case '.':
		if l.peekChar() == '.' {
			l.readChar()
			l.readChar()
			return token.Token{Type: token.RANGE, Literal: "..", Pos: pos}
		}
		l.readChar()
		return token.Token{Type: token.DOT, Literal: ".", Pos: pos}

	case ':':
		l.readChar()
		return token.Token{Type: token.COLON, Literal: ":", Pos: pos}

	case ',':
		l.readChar()
		return token.Token{Type: token.COMMA, Literal: ",", Pos: pos}

	case '!':
		l.readChar()
		return token.Token{Type: token.BANG, Literal: "!", Pos: pos}

	case '=':
		l.readChar()
		return token.Token{Type: token.ASSIGN, Literal: "=", Pos: pos}

	case '@':
		l.readChar()
		return token.Token{Type: token.AT, Literal: "@", Pos: pos}

	case '/':
		if l.regexMode {
			return l.readRegex()
		}
		l.readChar()
		return token.Token{Type: token.SLASH, Literal: "/", Pos: pos}

	case '(':
		l.parenDepth++
		l.readChar()
		return token.Token{Type: token.LPAREN, Literal: "(", Pos: pos}

	case ')':
		if l.parenDepth > 0 {
			l.parenDepth--
		}
		l.readChar()
		return token.Token{Type: token.RPAREN, Literal: ")", Pos: pos}

	case '{':
		l.braceDepth++
		l.readChar()
		return token.Token{Type: token.LBRACE, Literal: "{", Pos: pos}

	case '}':
		if l.braceDepth > 0 {
			l.braceDepth--
		}
		l.readChar()
		return token.Token{Type: token.RBRACE, Literal: "}", Pos: pos}

	case '[':
		l.bracketDepth++
		l.readChar()
		return token.Token{Type: token.LBRACKET, Literal: "[", Pos: pos}

	case ']':
		if l.bracketDepth > 0 {
			l.bracketDepth--
		}
		l.readChar()
		return token.Token{Type: token.RBRACKET, Literal: "]", Pos: pos}

	case '_':
		// Check if underscore is part of an identifier
		if isIdentContinue(l.peekChar()) {
			return l.readIdentifier()
		}
		l.readChar()
		return token.Token{Type: token.UNDERSCORE, Literal: "_", Pos: pos}

	case '"':
		return l.readString()

	default:
		if isDigit(l.ch) {
			return l.readNumber()
		}
		if isIdentStart(l.ch) {
			return l.readIdentifier()
		}
		ch := l.ch
		l.readChar()
		return token.Token{Type: token.ILLEGAL, Literal: string(ch), Pos: pos}
	}
}

func (l *Lexer) insideBrackets() bool {
	return l.parenDepth > 0 || l.braceDepth > 0 || l.bracketDepth > 0
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip spaces and tabs (not newlines — they are significant)
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
		}
		// Skip comments
		if l.ch == '#' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			continue
		}
		break
	}
}

func (l *Lexer) readIdentifier() token.Token {
	pos := l.curPos()
	start := l.pos

	// First character: [a-zA-Z_]
	l.readChar()

	// Continue: [a-zA-Z0-9_] and optionally hyphen followed by alphanumeric
	for {
		if isAlphaNumUnderscore(l.ch) {
			l.readChar()
			continue
		}
		// Hyphen: only part of identifier if followed by alphanumeric
		if l.ch == '-' && isAlphaNum(l.peekChar()) {
			l.readChar() // consume '-'
			l.readChar() // consume next char
			continue
		}
		break
	}

	lit := l.input[start:l.pos]
	tokType := token.LookupIdent(lit)

	return token.Token{Type: tokType, Literal: lit, Pos: pos}
}

func (l *Lexer) readNumber() token.Token {
	pos := l.curPos()
	start := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return token.Token{Type: token.INT, Literal: l.input[start:l.pos], Pos: pos}
}

func (l *Lexer) readString() token.Token {
	pos := l.curPos()
	l.readChar() // skip opening quote
	start := l.pos
	for l.ch != '"' && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar() // skip escape char
		}
		l.readChar()
	}
	lit := l.input[start:l.pos]
	if l.ch == '"' {
		l.readChar() // skip closing quote
	}
	return token.Token{Type: token.STRING, Literal: lit, Pos: pos}
}

func (l *Lexer) readRegex() token.Token {
	pos := l.curPos()
	l.readChar() // skip opening /
	start := l.pos
	for l.ch != '/' && l.ch != 0 && l.ch != '\n' {
		if l.ch == '\\' {
			l.readChar() // skip escape
		}
		l.readChar()
	}
	lit := l.input[start:l.pos]
	if l.ch == '/' {
		l.readChar() // skip closing /
	}
	return token.Token{Type: token.REGEX, Literal: lit, Pos: pos}
}

func isIdentStart(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || ch == '_'
}

func isIdentContinue(ch byte) bool {
	return isAlphaNumUnderscore(ch) || ch == '-'
}

func isAlphaNumUnderscore(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_'
}

func isAlphaNum(ch byte) bool {
	return unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch))
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
