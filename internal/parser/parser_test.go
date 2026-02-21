package parser

import (
	"testing"

	"github.com/polidog/reverhttp/internal/ast"
	"github.com/polidog/reverhttp/internal/lexer"
)

func parse(input string) *ast.File {
	l := lexer.New(input, "test.rever")
	p := New(l)
	return p.ParseFile()
}

func parseWithErrors(t *testing.T, input string) (*ast.File, []string) {
	t.Helper()
	l := lexer.New(input, "test.rever")
	p := New(l)
	f := p.ParseFile()
	return f, p.Errors()
}

func TestParseImport(t *testing.T) {
	input := `import fetch = github.com/reverhttp/std-fetch@0.1.0`
	f := parse(input)

	if len(f.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(f.Imports))
	}

	imp := f.Imports[0]
	if imp.Alias != "fetch" {
		t.Fatalf("expected alias 'fetch', got %q", imp.Alias)
	}
	if imp.Source != "github.com/reverhttp/std-fetch" {
		t.Fatalf("expected source 'github.com/reverhttp/std-fetch', got %q", imp.Source)
	}
	if imp.Version != "0.1.0" {
		t.Fatalf("expected version '0.1.0', got %q", imp.Version)
	}
}

func TestParseImportLocal(t *testing.T) {
	input := `import fetch = @/src/user/fetch.rever`
	f := parse(input)

	if len(f.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(f.Imports))
	}

	imp := f.Imports[0]
	if imp.Alias != "fetch" {
		t.Fatalf("expected alias 'fetch', got %q", imp.Alias)
	}
	if !imp.Local {
		t.Fatal("expected local import")
	}
	if imp.Source != "@/src/user/fetch.rever" {
		t.Fatalf("expected source '@/src/user/fetch.rever', got %q", imp.Source)
	}
}

func TestParseMultipleImports(t *testing.T) {
	input := `import fetch  = github.com/reverhttp/std-fetch@0.1.0
import create = github.com/reverhttp/std-create@0.1.0`
	f := parse(input)

	if len(f.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(f.Imports))
	}

	if f.Imports[0].Alias != "fetch" {
		t.Fatalf("expected alias 'fetch', got %q", f.Imports[0].Alias)
	}
	if f.Imports[1].Alias != "create" {
		t.Fatalf("expected alias 'create', got %q", f.Imports[1].Alias)
	}
}

func TestParseType(t *testing.T) {
	input := `type User {
  id: int
  name: string
  email: string
  created_at: datetime
}`
	f := parse(input)

	if len(f.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(f.Types))
	}

	td := f.Types[0]
	if td.Name != "User" {
		t.Fatalf("expected type name 'User', got %q", td.Name)
	}
	if len(td.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(td.Fields))
	}

	expected := []struct{ name, typ string }{
		{"id", "int"},
		{"name", "string"},
		{"email", "string"},
		{"created_at", "datetime"},
	}
	for i, exp := range expected {
		if td.Fields[i].Name != exp.name {
			t.Fatalf("field[%d] name: expected %q, got %q", i, exp.name, td.Fields[i].Name)
		}
		if td.Fields[i].TypeName != exp.typ {
			t.Fatalf("field[%d] type: expected %q, got %q", i, exp.typ, td.Fields[i].TypeName)
		}
	}
}

func TestParseSimpleGETRoute(t *testing.T) {
	input := `GET /users/{id}
  |> input(id: path.id)
  |> respond 200 { id: user.id }`

	f := parse(input)

	if len(f.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(f.Routes))
	}

	r := f.Routes[0]
	if r.Method != "GET" {
		t.Fatalf("expected method GET, got %q", r.Method)
	}
	if r.Path != "/users/{id}" {
		t.Fatalf("expected path '/users/{id}', got %q", r.Path)
	}
	if len(r.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(r.Steps))
	}

	// Check input step
	if r.Steps[0].Kind != ast.StepInput {
		t.Fatalf("expected StepInput, got %d", r.Steps[0].Kind)
	}
	if len(r.Steps[0].Input.Fields) != 1 {
		t.Fatalf("expected 1 input field, got %d", len(r.Steps[0].Input.Fields))
	}
	if r.Steps[0].Input.Fields[0].Name != "id" {
		t.Fatalf("expected input field name 'id', got %q", r.Steps[0].Input.Fields[0].Name)
	}
	if r.Steps[0].Input.Fields[0].From != "path.id" {
		t.Fatalf("expected input field from 'path.id', got %q", r.Steps[0].Input.Fields[0].From)
	}

	// Check respond step
	if r.Steps[1].Kind != ast.StepRespond {
		t.Fatalf("expected StepRespond, got %d", r.Steps[1].Kind)
	}
	if r.Steps[1].Respond.Status != "200" {
		t.Fatalf("expected status 200, got %q", r.Steps[1].Respond.Status)
	}
}

func TestParseValidate(t *testing.T) {
	input := `GET /test
  |> validate(id: int & min(1))  ~> 400 { error: "invalid id" }`

	f := parse(input)
	r := f.Routes[0]

	if len(r.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(r.Steps))
	}

	step := r.Steps[0]
	if step.Kind != ast.StepValidate {
		t.Fatalf("expected StepValidate, got %d", step.Kind)
	}

	rules := step.Validate.Rules
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Field != "id" {
		t.Fatalf("expected field 'id', got %q", rules[0].Field)
	}
	if len(rules[0].Constraints) != 2 {
		t.Fatalf("expected 2 constraints, got %d", len(rules[0].Constraints))
	}
	if rules[0].Constraints[0].Name != "int" {
		t.Fatalf("expected constraint 'int', got %q", rules[0].Constraints[0].Name)
	}
	if rules[0].Constraints[1].Name != "min" {
		t.Fatalf("expected constraint 'min', got %q", rules[0].Constraints[1].Name)
	}

	// Error flow
	if step.ErrorFlow == nil {
		t.Fatal("expected error flow")
	}
	if step.ErrorFlow.Status != "400" {
		t.Fatalf("expected error status 400, got %q", step.ErrorFlow.Status)
	}
}

func TestParseTransform(t *testing.T) {
	input := `GET /test
  |> transform(id: int(id), name: trim(name), email: lower(email))`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.Kind != ast.StepTransform {
		t.Fatalf("expected StepTransform, got %d", step.Kind)
	}

	fields := step.Transform.Fields
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	expected := []struct{ name, fn, from string }{
		{"id", "int", "id"},
		{"name", "trim", "name"},
		{"email", "lower", "email"},
	}
	for i, exp := range expected {
		if fields[i].Name != exp.name {
			t.Fatalf("field[%d] name: expected %q, got %q", i, exp.name, fields[i].Name)
		}
		if fields[i].Func != exp.fn {
			t.Fatalf("field[%d] func: expected %q, got %q", i, exp.fn, fields[i].Func)
		}
		if fields[i].From != exp.from {
			t.Fatalf("field[%d] from: expected %q, got %q", i, exp.from, fields[i].From)
		}
	}
}

func TestParseGuard(t *testing.T) {
	input := `GET /test
  |> guard !existing  ~> 409 { error: "already exists" }`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.Kind != ast.StepGuard {
		t.Fatalf("expected StepGuard, got %d", step.Kind)
	}
	if !step.Guard.Negated {
		t.Fatal("expected negated guard")
	}
	if step.Guard.Expr != "existing" {
		t.Fatalf("expected guard expr 'existing', got %q", step.Guard.Expr)
	}
	if step.ErrorFlow == nil || step.ErrorFlow.Status != "409" {
		t.Fatal("expected error flow with status 409")
	}
}

func TestParsePkgCallWithBind(t *testing.T) {
	input := `GET /test
  |> fetch(User, id) as user  ~> 404 { error: "not found" }`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.Kind != ast.StepPkgCall {
		t.Fatalf("expected StepPkgCall, got %d", step.Kind)
	}
	if step.PkgCall.Pkg != "fetch" {
		t.Fatalf("expected pkg 'fetch', got %q", step.PkgCall.Pkg)
	}
	if step.Bind != "user" {
		t.Fatalf("expected bind 'user', got %q", step.Bind)
	}
	if step.ErrorFlow == nil || step.ErrorFlow.Status != "404" {
		t.Fatal("expected error flow with status 404")
	}

	args := step.PkgCall.Args
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if !args[0].IsType || args[0].Value != "User" {
		t.Fatalf("expected first arg to be type 'User', got %+v", args[0])
	}
	if args[1].Value != "id" {
		t.Fatalf("expected second arg 'id', got %q", args[1].Value)
	}
}

func TestParseMatch(t *testing.T) {
	input := `GET /test
  |> match role {
       "user":  fetch(User, id)
       "admin": fetch(Admin, id)
       _:                          ~> 400 { error: "unknown" }
     } as account  ~> 404 { error: "not found" }`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.Kind != ast.StepMatch {
		t.Fatalf("expected StepMatch, got %d", step.Kind)
	}
	if step.Match.On != "role" {
		t.Fatalf("expected match on 'role', got %q", step.Match.On)
	}
	if len(step.Match.Arms) != 3 {
		t.Fatalf("expected 3 arms, got %d", len(step.Match.Arms))
	}
	if step.Bind != "account" {
		t.Fatalf("expected bind 'account', got %q", step.Bind)
	}
	if step.ErrorFlow == nil || step.ErrorFlow.Status != "404" {
		t.Fatal("expected error flow with status 404")
	}

	// Check first arm
	if step.Match.Arms[0].Pattern.Kind != ast.PatternLiteral || step.Match.Arms[0].Pattern.Value != "user" {
		t.Fatalf("expected literal pattern 'user', got %+v", step.Match.Arms[0].Pattern)
	}
	if step.Match.Arms[0].Step == nil || step.Match.Arms[0].Step.Pkg != "fetch" {
		t.Fatal("expected fetch step in first arm")
	}

	// Check default arm
	if !step.Match.Arms[2].IsDefault {
		t.Fatal("expected default arm")
	}
}

func TestParseRespondWithHeaders(t *testing.T) {
	input := `GET /test
  |> respond 301 with headers { location: "/new" }`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.Kind != ast.StepRespond {
		t.Fatalf("expected StepRespond, got %d", step.Kind)
	}
	if step.Respond.Status != "301" {
		t.Fatalf("expected status 301, got %q", step.Respond.Status)
	}
	if len(step.Respond.Headers) != 1 {
		t.Fatalf("expected 1 header, got %d", len(step.Respond.Headers))
	}
	if step.Respond.Headers[0].Key != "location" {
		t.Fatalf("expected header key 'location', got %q", step.Respond.Headers[0].Key)
	}
}

func TestParseDefaults(t *testing.T) {
	input := `defaults
  cors(origins: ["*"])
  auth(bearer)`

	f := parse(input)

	if f.Defaults == nil {
		t.Fatal("expected defaults block")
	}
	if len(f.Defaults.Directives) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(f.Defaults.Directives))
	}
	if f.Defaults.Directives[0].Name != "cors" {
		t.Fatalf("expected directive name 'cors', got %q", f.Defaults.Directives[0].Name)
	}
	if f.Defaults.Directives[1].Name != "auth" {
		t.Fatalf("expected directive name 'auth', got %q", f.Defaults.Directives[1].Name)
	}
}

func TestParseRouteWithDirectives(t *testing.T) {
	input := `GET /users/{id}
  cache(max-age: 3600, public, etag: hash(user))
  |> input(id: path.id)
  |> respond 200 { id: user.id }`

	f := parse(input)
	r := f.Routes[0]

	if len(r.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(r.Directives))
	}
	if r.Directives[0].Name != "cache" {
		t.Fatalf("expected directive 'cache', got %q", r.Directives[0].Name)
	}
}

func TestParseFullSpec6Example(t *testing.T) {
	input := `import fetch = github.com/reverhttp/std-fetch@0.1.0

GET /users/{id}
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }`

	f, errs := parseWithErrors(t, input)
	if len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}

	if len(f.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(f.Imports))
	}
	if len(f.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(f.Routes))
	}

	r := f.Routes[0]
	if r.Method != "GET" || r.Path != "/users/{id}" {
		t.Fatalf("expected GET /users/{id}, got %s %s", r.Method, r.Path)
	}
	if len(r.Steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(r.Steps))
	}
}

func TestParsePkgCallWithObject(t *testing.T) {
	input := `GET /test
  |> create(User, { name, email }) as user  ~> 500 { error: "failed" }`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	if step.PkgCall.Pkg != "create" {
		t.Fatalf("expected pkg 'create', got %q", step.PkgCall.Pkg)
	}
	args := step.PkgCall.Args
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if !args[0].IsType || args[0].Value != "User" {
		t.Fatalf("expected type arg 'User', got %+v", args[0])
	}
	if len(args[1].ObjectArgs) != 2 || args[1].ObjectArgs[0] != "name" || args[1].ObjectArgs[1] != "email" {
		t.Fatalf("expected object args [name, email], got %+v", args[1].ObjectArgs)
	}
}

func TestParseImportDelete(t *testing.T) {
	// "delete" is both a keyword and can be used as an import alias
	input := `import delete = github.com/reverhttp/std-delete@0.1.0`
	f := parse(input)

	if len(f.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(f.Imports))
	}
	if f.Imports[0].Alias != "delete" {
		t.Fatalf("expected alias 'delete', got %q", f.Imports[0].Alias)
	}
}

func TestParseMatchMultiValue(t *testing.T) {
	input := `GET /test
  |> match role {
       "user", "member": fetch(User, id)
       _:                                ~> 400 { error: "unknown" }
     } as account`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	arm := step.Match.Arms[0]
	if arm.Pattern.Kind != ast.PatternMulti {
		t.Fatalf("expected PatternMulti, got %d", arm.Pattern.Kind)
	}
	if len(arm.Pattern.Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(arm.Pattern.Values))
	}
}

func TestParseMatchRange(t *testing.T) {
	input := `GET /test
  |> match status_code {
       200..299: call(handleSuccess, response)
       _: call(handleError, response)
     } as result`

	f := parse(input)
	step := f.Routes[0].Steps[0]

	arm := step.Match.Arms[0]
	if arm.Pattern.Kind != ast.PatternRange {
		t.Fatalf("expected PatternRange, got %d", arm.Pattern.Kind)
	}
	if arm.Pattern.RangeMin != "200" || arm.Pattern.RangeMax != "299" {
		t.Fatalf("expected range 200..299, got %s..%s", arm.Pattern.RangeMin, arm.Pattern.RangeMax)
	}
}

func TestParseCorsNone(t *testing.T) {
	input := `GET /test
  cors(none)
  |> respond 200 { status: "ok" }`

	f := parse(input)
	r := f.Routes[0]

	if len(r.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(r.Directives))
	}
	d := r.Directives[0]
	if d.Name != "cors" {
		t.Fatalf("expected 'cors', got %q", d.Name)
	}
	if len(d.Args) != 1 || d.Args[0].Name != "none" {
		t.Fatalf("expected none arg, got %+v", d.Args)
	}
}

func TestParseAuthWithBind(t *testing.T) {
	input := `GET /admin
  auth(bearer, roles: ["admin"]) as current_user
  |> respond 200 { ok: "true" }`

	f := parse(input)
	r := f.Routes[0]

	if len(r.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(r.Directives))
	}
	d := r.Directives[0]
	if d.Name != "auth" {
		t.Fatalf("expected 'auth', got %q", d.Name)
	}
	if d.Bind != "current_user" {
		t.Fatalf("expected bind 'current_user', got %q", d.Bind)
	}
}
