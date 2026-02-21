package token

type Type int

const (
	// Special
	ILLEGAL Type = iota
	EOF
	NEWLINE

	// Literals
	IDENT  // identifier (including hyphenated like redis-cache)
	INT    // 123
	STRING // "hello"
	REGEX  // /pattern/

	// Operators and delimiters
	PIPE      // |>
	ERROR     // ~>
	AMPERSAND // &
	RANGE     // ..
	COLON     // :
	COMMA     // ,
	DOT       // .
	BANG      // !
	ASSIGN    // =
	AT        // @
	SLASH     // /

	LPAREN   // (
	RPAREN   // )
	LBRACE   // {
	RBRACE   // }
	LBRACKET // [
	RBRACKET // ]

	UNDERSCORE // _

	// Keywords
	IMPORT
	TYPE
	DEFAULTS
	AS
	MATCH
	GUARD
	RESPOND
	INPUT
	VALIDATE
	TRANSFORM
	WITH
	HEADERS
	CACHE
	CORS
	AUTH
	NONE

	// HTTP methods
	GET
	POST
	PUT
	DELETE
	PATCH
	HEAD
	OPTIONS
)

var typeNames = map[Type]string{
	ILLEGAL:    "ILLEGAL",
	EOF:        "EOF",
	NEWLINE:    "NEWLINE",
	IDENT:      "IDENT",
	INT:        "INT",
	STRING:     "STRING",
	REGEX:      "REGEX",
	PIPE:       "|>",
	ERROR:      "~>",
	AMPERSAND:  "&",
	RANGE:      "..",
	COLON:      ":",
	COMMA:      ",",
	DOT:        ".",
	BANG:       "!",
	ASSIGN:     "=",
	AT:         "@",
	SLASH:      "/",
	LPAREN:     "(",
	RPAREN:     ")",
	LBRACE:     "{",
	RBRACE:     "}",
	LBRACKET:   "[",
	RBRACKET:   "]",
	UNDERSCORE: "_",
	IMPORT:     "import",
	TYPE:       "type",
	DEFAULTS:   "defaults",
	AS:         "as",
	MATCH:      "match",
	GUARD:      "guard",
	RESPOND:    "respond",
	INPUT:      "input",
	VALIDATE:   "validate",
	TRANSFORM:  "transform",
	WITH:       "with",
	HEADERS:    "headers",
	CACHE:      "cache",
	CORS:       "cors",
	AUTH:       "auth",
	NONE:       "none",
	GET:        "GET",
	POST:       "POST",
	PUT:        "PUT",
	DELETE:     "DELETE",
	PATCH:      "PATCH",
	HEAD:       "HEAD",
	OPTIONS:    "OPTIONS",
}

func (t Type) String() string {
	if s, ok := typeNames[t]; ok {
		return s
	}
	return "UNKNOWN"
}

var keywords = map[string]Type{
	"import":    IMPORT,
	"type":      TYPE,
	"defaults":  DEFAULTS,
	"as":        AS,
	"match":     MATCH,
	"guard":     GUARD,
	"respond":   RESPOND,
	"input":     INPUT,
	"validate":  VALIDATE,
	"transform": TRANSFORM,
	"with":      WITH,
	"headers":   HEADERS,
	"cache":     CACHE,
	"cors":      CORS,
	"auth":      AUTH,
	"none":      NONE,
	"GET":       GET,
	"POST":      POST,
	"PUT":       PUT,
	"DELETE":    DELETE,
	"PATCH":     PATCH,
	"HEAD":      HEAD,
	"OPTIONS":   OPTIONS,
}

// LookupIdent returns the keyword token type for ident, or IDENT if not a keyword.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// IsHTTPMethod returns true if the token type is an HTTP method keyword.
func IsHTTPMethod(t Type) bool {
	switch t {
	case GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS:
		return true
	}
	return false
}

// Position represents a source location.
type Position struct {
	File   string
	Line   int
	Column int
}

// Token represents a lexical token with its position and literal value.
type Token struct {
	Type    Type
	Literal string
	Pos     Position
}
