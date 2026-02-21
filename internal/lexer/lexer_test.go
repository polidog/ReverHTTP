package lexer

import (
	"testing"

	"github.com/polidog/reverhttp/internal/token"
)

func TestNextToken_Operators(t *testing.T) {
	input := `|> ~> & .. : , . ! = @`
	l := New(input, "test")

	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.PIPE, "|>"},
		{token.ERROR, "~>"},
		{token.AMPERSAND, "&"},
		{token.RANGE, ".."},
		{token.COLON, ":"},
		{token.COMMA, ","},
		{token.DOT, "."},
		{token.BANG, "!"},
		{token.ASSIGN, "="},
		{token.AT, "@"},
		{token.EOF, ""},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ {
			t.Fatalf("test[%d] - type wrong. expected=%q, got=%q", i, exp.typ, tok.Type)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q", i, exp.lit, tok.Literal)
		}
	}
}

func TestNextToken_Brackets(t *testing.T) {
	input := `( ) { } [ ]`
	l := New(input, "test")

	expected := []token.Type{
		token.LPAREN, token.RPAREN,
		token.LBRACE, token.RBRACE,
		token.LBRACKET, token.RBRACKET,
		token.EOF,
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - type wrong. expected=%q, got=%q", i, exp, tok.Type)
		}
	}
}

func TestNextToken_Keywords(t *testing.T) {
	input := `import type defaults as match guard respond input validate transform with headers cache cors auth none`
	l := New(input, "test")

	expected := []token.Type{
		token.IMPORT, token.TYPE, token.DEFAULTS, token.AS,
		token.MATCH, token.GUARD, token.RESPOND,
		token.INPUT, token.VALIDATE, token.TRANSFORM,
		token.WITH, token.HEADERS, token.CACHE, token.CORS, token.AUTH, token.NONE,
		token.EOF,
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - type wrong. expected=%q, got=%q (literal=%q)", i, exp, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_HTTPMethods(t *testing.T) {
	input := `GET POST PUT DELETE PATCH HEAD OPTIONS`
	l := New(input, "test")

	expected := []token.Type{
		token.GET, token.POST, token.PUT, token.DELETE,
		token.PATCH, token.HEAD, token.OPTIONS,
		token.EOF,
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp {
			t.Fatalf("test[%d] - type wrong. expected=%q, got=%q", i, exp, tok.Type)
		}
	}
}

func TestNextToken_HyphenatedIdent(t *testing.T) {
	input := `redis-cache max-age x-role api-key expose-headers`
	l := New(input, "test")

	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.IDENT, "redis-cache"},
		{token.IDENT, "max-age"},
		{token.IDENT, "x-role"},
		{token.IDENT, "api-key"},
		{token.IDENT, "expose-headers"},
		{token.EOF, ""},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ {
			t.Fatalf("test[%d] - type wrong. expected=%q, got=%q", i, exp.typ, tok.Type)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q", i, exp.lit, tok.Literal)
		}
	}
}

func TestNextToken_StringLiteral(t *testing.T) {
	input := `"hello world" "invalid id"`
	l := New(input, "test")

	tok := l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "hello world" {
		t.Fatalf("expected STRING 'hello world', got %s %q", tok.Type, tok.Literal)
	}

	tok = l.NextToken()
	if tok.Type != token.STRING || tok.Literal != "invalid id" {
		t.Fatalf("expected STRING 'invalid id', got %s %q", tok.Type, tok.Literal)
	}
}

func TestNextToken_IntLiteral(t *testing.T) {
	input := `123 400 200`
	l := New(input, "test")

	for _, exp := range []string{"123", "400", "200"} {
		tok := l.NextToken()
		if tok.Type != token.INT || tok.Literal != exp {
			t.Fatalf("expected INT %q, got %s %q", exp, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_RegexMode(t *testing.T) {
	l := New(`/^admin/`, "test")
	l.SetRegexMode(true)

	tok := l.NextToken()
	if tok.Type != token.REGEX || tok.Literal != "^admin" {
		t.Fatalf("expected REGEX '^admin', got %s %q", tok.Type, tok.Literal)
	}
}

func TestNextToken_SlashWithoutRegexMode(t *testing.T) {
	l := New(`/users`, "test")

	tok := l.NextToken()
	if tok.Type != token.SLASH {
		t.Fatalf("expected SLASH, got %s", tok.Type)
	}
}

func TestNextToken_Comment(t *testing.T) {
	input := "# this is a comment\nGET"
	l := New(input, "test")

	tok := l.NextToken()
	if tok.Type != token.NEWLINE {
		t.Fatalf("expected NEWLINE after comment, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != token.GET {
		t.Fatalf("expected GET after comment, got %s %q", tok.Type, tok.Literal)
	}
}

func TestNextToken_NewlineSuppression(t *testing.T) {
	// Newlines inside parentheses should be suppressed
	input := "(\n\n)"
	l := New(input, "test")

	tok := l.NextToken()
	if tok.Type != token.LPAREN {
		t.Fatalf("expected LPAREN, got %s", tok.Type)
	}
	tok = l.NextToken()
	if tok.Type != token.RPAREN {
		t.Fatalf("expected RPAREN (newlines suppressed), got %s", tok.Type)
	}
}

func TestNextToken_Underscore(t *testing.T) {
	input := `_`
	l := New(input, "test")

	tok := l.NextToken()
	if tok.Type != token.UNDERSCORE {
		t.Fatalf("expected UNDERSCORE, got %s %q", tok.Type, tok.Literal)
	}
}

func TestNextToken_FullRoute(t *testing.T) {
	input := `GET /users/{id}
  |> input(id: path.id)
  |> respond 200 { id: user.id }`
	l := New(input, "test")

	expected := []struct {
		typ token.Type
		lit string
	}{
		{token.GET, "GET"},
		{token.SLASH, "/"},
		{token.IDENT, "users"},
		{token.SLASH, "/"},
		{token.LBRACE, "{"},
		{token.IDENT, "id"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
		{token.PIPE, "|>"},
		{token.INPUT, "input"},
		{token.LPAREN, "("},
		{token.IDENT, "id"},
		{token.COLON, ":"},
		{token.IDENT, "path"},
		{token.DOT, "."},
		{token.IDENT, "id"},
		{token.RPAREN, ")"},
		{token.NEWLINE, "\n"},
		{token.PIPE, "|>"},
		{token.RESPOND, "respond"},
		{token.INT, "200"},
		{token.LBRACE, "{"},
		{token.IDENT, "id"},
		{token.COLON, ":"},
		{token.IDENT, "user"},
		{token.DOT, "."},
		{token.IDENT, "id"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	for i, exp := range expected {
		tok := l.NextToken()
		if tok.Type != exp.typ {
			t.Fatalf("test[%d] - type wrong. expected=%s, got=%s (literal=%q)", i, exp.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != exp.lit {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q", i, exp.lit, tok.Literal)
		}
	}
}

func TestNextToken_Position(t *testing.T) {
	input := "GET\nimport"
	l := New(input, "test.rever")

	tok := l.NextToken()
	if tok.Pos.Line != 1 || tok.Pos.File != "test.rever" {
		t.Fatalf("expected line 1 file test.rever, got line %d file %s", tok.Pos.Line, tok.Pos.File)
	}

	l.NextToken() // NEWLINE

	tok = l.NextToken()
	if tok.Pos.Line != 2 {
		t.Fatalf("expected line 2, got line %d", tok.Pos.Line)
	}
}
