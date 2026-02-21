package gen

import (
	"strconv"
	"strings"

	"github.com/polidog/reverhttp/internal/ast"
	"github.com/polidog/reverhttp/internal/ir"
)

// Known type names for cast vs fn distinction in transforms.
var typeNames = map[string]bool{
	"int":      true,
	"string":   true,
	"bool":     true,
	"float":    true,
	"datetime": true,
}

// Generate converts an AST File to the IR Root.
func Generate(file *ast.File) *ir.Root {
	root := &ir.Root{
		Version: "0.1",
	}

	// Imports
	if len(file.Imports) > 0 {
		root.Imports = make(map[string]*ir.Import)
		for _, imp := range file.Imports {
			entry := &ir.Import{Source: imp.Source}
			if imp.Local {
				entry.Local = true
			} else {
				entry.Version = imp.Version
			}
			root.Imports[imp.Alias] = entry
		}
	}

	// Types
	if len(file.Types) > 0 {
		root.Types = make(map[string]ir.TypeFields)
		for _, td := range file.Types {
			fields := make(ir.TypeFields)
			for _, f := range td.Fields {
				fields[f.Name] = f.TypeName
			}
			root.Types[td.Name] = fields
		}
	}

	// Defaults
	if file.Defaults != nil {
		root.Defaults = genDefaults(file.Defaults)
	}

	// Routes
	for _, route := range file.Routes {
		root.Routes = append(root.Routes, genRoute(route))
	}

	return root
}

func genDefaults(block *ast.DefaultsBlock) *ir.Defaults {
	d := &ir.Defaults{}
	for _, dir := range block.Directives {
		switch dir.Name {
		case "cache":
			d.Cache = genCache(dir)
		case "cors":
			d.CORS = genCORS(dir)
		case "auth":
			d.Auth = genAuth(dir)
		}
	}
	return d
}

func genRoute(route *ast.Route) *ir.Route {
	r := &ir.Route{
		RouteInfo: &ir.RouteInfo{
			Method: route.Method,
			Path:   route.Path,
		},
	}

	// Directives
	for _, dir := range route.Directives {
		switch dir.Name {
		case "cache":
			r.Cache = genCache(dir)
		case "cors":
			if isNoneDirective(dir) {
				// cors(none) → "cors": null — use a special marker
				r.CORS = nil // We'll handle this in JSON marshaling
				// Actually, we need a way to distinguish "no cors" from "cors not set"
				// We'll use a sentinel: set CORS to explicit null
				r.CORS = (*ir.CORS)(nil) // explicit nil pointer
			} else {
				r.CORS = genCORS(dir)
			}
		case "auth":
			if isNoneDirective(dir) {
				// auth(none) → omit
			} else {
				r.Auth = genAuth(dir)
			}
		}
	}

	// Pipeline steps
	var processSteps []interface{}

	for _, step := range route.Steps {
		switch step.Kind {
		case ast.StepInput:
			r.Input = genInput(step.Input)

		case ast.StepValidate:
			r.Validate = genValidate(step)

		case ast.StepTransform:
			r.TransformIn = genTransform(step.Transform)

		case ast.StepGuard:
			gs := genGuard(step)
			processSteps = append(processSteps, gs)

		case ast.StepMatch:
			ms := genMatch(step)
			processSteps = append(processSteps, ms)

		case ast.StepPkgCall:
			ps := genPkgCall(step)
			processSteps = append(processSteps, ps)

		case ast.StepRespond:
			r.Output = genRespond(step.Respond)
		}
	}

	if len(processSteps) > 0 {
		r.Process = &ir.Process{Steps: processSteps}
	}

	return r
}

func genInput(input *ast.InputStep) map[string]*ir.Input {
	if input == nil {
		return nil
	}
	result := make(map[string]*ir.Input)
	for _, f := range input.Fields {
		result[f.Name] = &ir.Input{From: f.From}
	}
	return result
}

func genValidate(step *ast.PipelineStep) *ir.Validate {
	if step.Validate == nil {
		return nil
	}

	v := &ir.Validate{
		Rules: make(map[string]*ir.ValidateRule),
	}

	for _, rule := range step.Validate.Rules {
		vr := &ir.ValidateRule{}
		for _, c := range rule.Constraints {
			switch c.Name {
			case "int", "string", "bool", "float", "datetime":
				vr.Type = c.Name
			case "min":
				if len(c.Args) > 0 {
					if val, err := strconv.Atoi(c.Args[0].IntVal); err == nil {
						vr.Min = intPtr(val)
					}
				}
			case "max":
				if len(c.Args) > 0 {
					if val, err := strconv.Atoi(c.Args[0].IntVal); err == nil {
						vr.Max = intPtr(val)
					}
				}
			case "format":
				if len(c.Args) > 0 {
					vr.Format = c.Args[0].StrVal
				}
			}
		}
		v.Rules[rule.Field] = vr
	}

	if step.ErrorFlow != nil {
		v.Error = genErrorResponse(step.ErrorFlow)
	}

	return v
}

func genTransform(t *ast.TransformStep) map[string]*ir.Transform {
	if t == nil {
		return nil
	}
	result := make(map[string]*ir.Transform)
	for _, f := range t.Fields {
		tr := &ir.Transform{From: f.From}
		if typeNames[f.Func] {
			tr.Cast = f.Func
		} else {
			tr.Fn = f.Func
		}
		result[f.Name] = tr
	}
	return result
}

func genGuard(step *ast.PipelineStep) *ir.GuardStep {
	gs := &ir.GuardStep{}
	if step.Guard.Negated {
		gs.Guard = map[string]string{"not": step.Guard.Expr}
	} else {
		gs.Guard = step.Guard.Expr
	}
	if step.ErrorFlow != nil {
		gs.Error = genErrorResponse(step.ErrorFlow)
	}
	return gs
}

func genMatch(step *ast.PipelineStep) *ir.MatchProcessStep {
	m := step.Match
	ms := &ir.MatchProcessStep{
		Bind: step.Bind,
		Match: &ir.MatchBlock{
			On: m.On,
		},
	}

	for _, arm := range m.Arms {
		if arm.IsDefault {
			if arm.ErrorOnly && arm.ErrorFlow != nil {
				ms.Match.Default = &ir.MatchDefaultError{
					Error: genErrorResponse(arm.ErrorFlow),
				}
			} else if arm.VarRef != "" {
				// Default arm with variable reference
				ms.Match.Default = map[string]string{"ref": arm.VarRef}
			} else if arm.Step != nil {
				irArm := genMatchArmStep(arm)
				ms.Match.Default = irArm
			}
			continue
		}

		irArm := &ir.MatchArm{
			Pattern: genPattern(arm.Pattern),
		}

		if arm.Step != nil {
			irArm.Use = arm.Step.Pkg
			irArm.Input = genPkgInput(arm.Step)
		} else if arm.VarRef != "" {
			irArm.Ref = arm.VarRef
		}

		if arm.ErrorFlow != nil {
			irArm.Error = genErrorResponse(arm.ErrorFlow)
		}

		ms.Match.Arms = append(ms.Match.Arms, irArm)
	}

	if step.ErrorFlow != nil {
		ms.Error = genErrorResponse(step.ErrorFlow)
	}

	return ms
}

func genMatchArmStep(arm *ast.MatchArm) *ir.MatchArm {
	irArm := &ir.MatchArm{}
	if arm.Step != nil {
		irArm.Use = arm.Step.Pkg
		irArm.Input = genPkgInput(arm.Step)
	}
	if arm.ErrorFlow != nil {
		irArm.Error = genErrorResponse(arm.ErrorFlow)
	}
	return irArm
}

func genPattern(p ast.Pattern) interface{} {
	switch p.Kind {
	case ast.PatternLiteral:
		// Try to parse as int
		if val, err := strconv.Atoi(p.Value); err == nil {
			return &ir.PatternValue{Value: val}
		}
		// Bool
		if p.Value == "true" || p.Value == "false" {
			return &ir.PatternValue{Value: p.Value == "true"}
		}
		// String (could be "null")
		if p.Value == "null" {
			return &ir.PatternValue{Value: nil}
		}
		return &ir.PatternValue{Value: p.Value}

	case ast.PatternMulti:
		vals := make([]interface{}, len(p.Values))
		for i, v := range p.Values {
			vals[i] = v
		}
		return &ir.PatternIn{In: vals}

	case ast.PatternRange:
		min, _ := strconv.Atoi(p.RangeMin)
		max, _ := strconv.Atoi(p.RangeMax)
		return &ir.PatternRange{Range: &ir.RangeValue{Min: min, Max: max}}

	case ast.PatternRegex:
		return &ir.PatternRegex{Regex: p.Regex}

	default:
		return nil
	}
}

func genPkgCall(step *ast.PipelineStep) *ir.PkgStep {
	ps := &ir.PkgStep{
		Bind:  step.Bind,
		Use:   step.PkgCall.Pkg,
		Input: genPkgInput(step.PkgCall),
	}
	if step.ErrorFlow != nil {
		ps.Error = genErrorResponse(step.ErrorFlow)
	}
	return ps
}

func genPkgInput(call *ast.PkgCallStep) map[string]interface{} {
	input := make(map[string]interface{})
	for _, arg := range call.Args {
		if arg.Name != "" {
			input[arg.Name] = arg.Value
		} else if arg.IsType {
			input["type"] = arg.Value
		} else if len(arg.ObjectArgs) > 0 {
			data := make(map[string]string)
			for _, k := range arg.ObjectArgs {
				data[k] = k
			}
			input["data"] = data
		} else if arg.Value != "" {
			// Positional args after the type: use common convention
			// If there's already a "type", this is likely the ID or other param
			if _, hasType := input["type"]; hasType {
				// Determine the key: for single values, use "id" as convention
				// But we need to be smarter here
				input["id"] = arg.Value
			} else {
				input[arg.Value] = arg.Value
			}
		}
	}
	return input
}

func genRespond(r *ast.RespondStep) *ir.Output {
	if r == nil {
		return nil
	}
	status, _ := strconv.Atoi(r.Status)
	o := &ir.Output{Status: status}

	if len(r.Body) > 0 {
		o.Body = make(map[string]string)
		for _, f := range r.Body {
			o.Body[f.Key] = f.Value
		}
	}

	if len(r.Headers) > 0 {
		o.Headers = make(map[string]string)
		for _, f := range r.Headers {
			o.Headers[f.Key] = f.Value
		}
	}

	return o
}

func genErrorResponse(ef *ast.ErrorFlow) *ir.ErrorResponse {
	if ef == nil {
		return nil
	}
	status, _ := strconv.Atoi(ef.Status)
	er := &ir.ErrorResponse{Status: status}
	if len(ef.Body) > 0 {
		er.Body = make(map[string]string)
		for _, f := range ef.Body {
			er.Body[f.Key] = f.Value
		}
	}
	return er
}

func genCache(dir *ast.Directive) *ir.Cache {
	c := &ir.Cache{}
	for _, arg := range dir.Args {
		switch arg.Name {
		case "max-age":
			if v, err := strconv.Atoi(arg.Value.IntVal); err == nil {
				c.MaxAge = intPtr(v)
			}
		case "s-maxage":
			if v, err := strconv.Atoi(arg.Value.IntVal); err == nil {
				c.SMaxAge = intPtr(v)
			}
		case "etag":
			c.ETag = genCacheExpr(arg.Value)
		case "last-modified":
			c.LastModified = arg.Value.StrVal
		case "vary":
			c.Vary = arg.Value.ListVal
		case "":
			// Positional flags
			switch arg.Value.StrVal {
			case "public":
				c.Visibility = "public"
			case "private":
				c.Visibility = "private"
			case "no-cache":
				c.NoCache = boolPtr(true)
			case "no-store":
				c.NoStore = boolPtr(true)
			}
		}
	}
	return c
}

func genCacheExpr(expr ast.Expr) interface{} {
	if expr.Kind == ast.ExprFuncCall {
		// Parse "hash(user)" → {fn: "hash", from: "user"}
		s := expr.StrVal
		if idx := strings.Index(s, "("); idx != -1 {
			fn := s[:idx]
			from := strings.TrimSuffix(s[idx+1:], ")")
			return &ir.ETagFn{Fn: fn, From: from}
		}
	}
	return expr.StrVal
}

func genCORS(dir *ast.Directive) *ir.CORS {
	c := &ir.CORS{}
	for _, arg := range dir.Args {
		switch arg.Name {
		case "origins":
			c.Origins = arg.Value.ListVal
		case "methods":
			c.Methods = arg.Value.ListVal
		case "headers":
			c.Headers = arg.Value.ListVal
		case "expose-headers":
			c.ExposeHeaders = arg.Value.ListVal
		case "max-age":
			if v, err := strconv.Atoi(arg.Value.IntVal); err == nil {
				c.MaxAge = intPtr(v)
			}
		case "":
			if arg.Value.StrVal == "credentials" {
				c.Credentials = boolPtr(true)
			}
		}
	}
	return c
}

func genAuth(dir *ast.Directive) *ir.Auth {
	a := &ir.Auth{}
	for _, arg := range dir.Args {
		switch arg.Name {
		case "roles":
			a.Roles = arg.Value.ListVal
		case "permissions":
			a.Permissions = arg.Value.ListVal
		case "":
			// First positional arg is the method
			if a.Method == "" {
				a.Method = arg.Value.StrVal
			}
		}
	}
	if dir.Bind != "" {
		a.Bind = dir.Bind
	}
	return a
}

func isNoneDirective(dir *ast.Directive) bool {
	for _, arg := range dir.Args {
		if arg.Name == "none" {
			return true
		}
	}
	return false
}

func intPtr(v int) *int    { return &v }
func boolPtr(v bool) *bool { return &v }
