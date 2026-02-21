package gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/polidog/reverhttp/internal/ir"
	"github.com/polidog/reverhttp/internal/lexer"
	"github.com/polidog/reverhttp/internal/parser"
)

func TestGenerateSpec6_GET(t *testing.T) {
	input := `import fetch = github.com/reverhttp/std-fetch@0.1.0

GET /users/{id}
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }`

	root := parseAndGenerate(input)

	// Check version
	if root.Version != "0.1" {
		t.Fatalf("expected version 0.1, got %q", root.Version)
	}

	// Check imports
	if len(root.Imports) != 1 {
		t.Fatalf("expected 1 import, got %d", len(root.Imports))
	}
	fetch := root.Imports["fetch"]
	if fetch.Source != "github.com/reverhttp/std-fetch" {
		t.Fatalf("expected source 'github.com/reverhttp/std-fetch', got %q", fetch.Source)
	}
	if fetch.Version != "0.1.0" {
		t.Fatalf("expected version '0.1.0', got %q", fetch.Version)
	}

	// Check route
	if len(root.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(root.Routes))
	}
	r := root.Routes[0]
	if r.RouteInfo.Method != "GET" || r.RouteInfo.Path != "/users/{id}" {
		t.Fatalf("expected GET /users/{id}, got %s %s", r.RouteInfo.Method, r.RouteInfo.Path)
	}

	// Check input
	if r.Input["id"].From != "path.id" {
		t.Fatalf("expected input id from 'path.id', got %q", r.Input["id"].From)
	}

	// Check validate
	if r.Validate == nil {
		t.Fatal("expected validate")
	}
	idRule := r.Validate.Rules["id"]
	if idRule.Type != "int" {
		t.Fatalf("expected validate type 'int', got %q", idRule.Type)
	}
	if *idRule.Min != 1 {
		t.Fatalf("expected validate min 1, got %d", *idRule.Min)
	}
	if r.Validate.Error.Status != 400 {
		t.Fatalf("expected validate error status 400, got %d", r.Validate.Error.Status)
	}

	// Check transform
	if r.TransformIn["id"].Cast != "int" {
		t.Fatalf("expected transform cast 'int', got %q", r.TransformIn["id"].Cast)
	}

	// Check process
	if r.Process == nil || len(r.Process.Steps) != 1 {
		t.Fatalf("expected 1 process step, got %v", r.Process)
	}

	// Check output
	if r.Output.Status != 200 {
		t.Fatalf("expected output status 200, got %d", r.Output.Status)
	}
	if r.Output.Body["id"] != "user.id" {
		t.Fatalf("expected body id 'user.id', got %q", r.Output.Body["id"])
	}
}

func TestGenerateSpec7_POST(t *testing.T) {
	input := `import fetch  = github.com/reverhttp/std-fetch@0.1.0
import create = github.com/reverhttp/std-create@0.1.0

POST /users
  |> input(name: body.name, email: body.email)
  |> validate(
       name: string & min(1) & max(100),
       email: string & format(email)
     )                                   ~> 400 { error: "validation failed", details: errors }
  |> transform(name: trim(name), email: lower(email))
  |> fetch(User, email: email) as existing
  |> guard !existing                     ~> 409 { error: "email already taken" }
  |> create(User, { name, email }) as user ~> 500 { error: "creation failed" }
  |> respond 201 { id: user.id, name: user.name, email: user.email }`

	root := parseAndGenerate(input)

	if len(root.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(root.Imports))
	}

	r := root.Routes[0]

	// Check transform: trim should be fn, not cast
	if r.TransformIn["name"].Fn != "trim" {
		t.Fatalf("expected transform fn 'trim', got fn=%q cast=%q", r.TransformIn["name"].Fn, r.TransformIn["name"].Cast)
	}
	if r.TransformIn["email"].Fn != "lower" {
		t.Fatalf("expected transform fn 'lower', got %q", r.TransformIn["email"].Fn)
	}

	// Check process steps count: fetch + guard + create = 3
	if r.Process == nil || len(r.Process.Steps) != 3 {
		t.Fatalf("expected 3 process steps, got %d", len(r.Process.Steps))
	}

	// Check output status
	if r.Output.Status != 201 {
		t.Fatalf("expected status 201, got %d", r.Output.Status)
	}
}

func TestGenerateRespondHeaders(t *testing.T) {
	input := `GET /test
  |> respond 301 with headers { location: "/new" }`

	root := parseAndGenerate(input)
	r := root.Routes[0]

	if r.Output.Status != 301 {
		t.Fatalf("expected status 301, got %d", r.Output.Status)
	}
	if r.Output.Headers["location"] != "/new" {
		t.Fatalf("expected header location '/new', got %q", r.Output.Headers["location"])
	}
}

func TestGenerateRespondNoBody(t *testing.T) {
	input := `GET /test
  |> respond 204`

	root := parseAndGenerate(input)
	r := root.Routes[0]

	if r.Output.Status != 204 {
		t.Fatalf("expected status 204, got %d", r.Output.Status)
	}
	if r.Output.Body != nil {
		t.Fatalf("expected no body, got %v", r.Output.Body)
	}
}

func TestGenerateTypes(t *testing.T) {
	input := `type User {
  id: int
  name: string
  email: string
}`

	root := parseAndGenerate(input)

	if root.Types == nil || len(root.Types) != 1 {
		t.Fatalf("expected 1 type, got %v", root.Types)
	}
	user := root.Types["User"]
	if user["id"] != "int" {
		t.Fatalf("expected User.id type 'int', got %q", user["id"])
	}
}

func TestGenerateJSONOutput(t *testing.T) {
	input := `import fetch = github.com/reverhttp/std-fetch@0.1.0

GET /users/{id}
  |> input(id: path.id)
  |> validate(id: int & min(1))          ~> 400 { error: "invalid id" }
  |> transform(id: int(id))
  |> fetch(User, id) as user             ~> 404 { error: "user not found" }
  |> respond 200 { id: user.id, name: user.name, email: user.email }`

	root := parseAndGenerate(input)

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	// Verify it's valid JSON
	var check map[string]interface{}
	if err := json.Unmarshal(data, &check); err != nil {
		t.Fatalf("JSON unmarshal error: %v\nJSON:\n%s", err, string(data))
	}

	// Verify key sections exist
	if _, ok := check["version"]; !ok {
		t.Fatal("missing 'version' in JSON output")
	}
	if _, ok := check["imports"]; !ok {
		t.Fatal("missing 'imports' in JSON output")
	}
	if _, ok := check["routes"]; !ok {
		t.Fatal("missing 'routes' in JSON output")
	}
}

func TestGoldenFile(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Skipf("no testdata directory: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".rever" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			reverFile := filepath.Join(testdataDir, entry.Name())
			baseName := entry.Name()[:len(entry.Name())-len(".rever")]
			expectedFile := filepath.Join(testdataDir, "expected", baseName+".json")

			input, err := os.ReadFile(reverFile)
			if err != nil {
				t.Fatalf("failed to read %s: %v", reverFile, err)
			}

			expectedData, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Skipf("no expected file %s: %v", expectedFile, err)
			}

			root := parseAndGenerate(string(input))
			actual, err := json.MarshalIndent(root, "", "  ")
			if err != nil {
				t.Fatalf("JSON marshal error: %v", err)
			}

			// Normalize both for comparison
			var expectedObj, actualObj interface{}
			if err := json.Unmarshal(expectedData, &expectedObj); err != nil {
				t.Fatalf("failed to parse expected JSON: %v", err)
			}
			if err := json.Unmarshal(actual, &actualObj); err != nil {
				t.Fatalf("failed to parse actual JSON: %v", err)
			}

			expectedNorm, _ := json.MarshalIndent(expectedObj, "", "  ")
			actualNorm, _ := json.MarshalIndent(actualObj, "", "  ")

			if string(expectedNorm) != string(actualNorm) {
				t.Errorf("output mismatch for %s\n--- expected ---\n%s\n--- actual ---\n%s",
					entry.Name(), string(expectedNorm), string(actualNorm))
			}
		})
	}
}

func parseAndGenerate(input string) *ir.Root {
	l := lexer.New(input, "test.rever")
	p := parser.New(l)
	file := p.ParseFile()
	return Generate(file)
}
