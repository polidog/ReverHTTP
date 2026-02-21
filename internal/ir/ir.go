package ir

// Root is the top-level IR structure for a ReverHTTP application.
type Root struct {
	Version  string                `json:"version"`
	Imports  map[string]*Import    `json:"imports,omitempty"`
	Types    map[string]TypeFields `json:"types,omitempty"`
	Defaults *Defaults             `json:"defaults,omitempty"`
	Routes   []*Route              `json:"routes"`
}

// TypeFields maps field names to type names.
type TypeFields map[string]string

// Import represents an imported package.
type Import struct {
	Source  string `json:"source"`
	Version string `json:"version,omitempty"`
	Local   bool   `json:"local,omitempty"`
}

// Defaults represents default directives applied to all routes.
type Defaults struct {
	Cache *Cache `json:"cache,omitempty"`
	CORS  *CORS  `json:"cors,omitempty"`
	Auth  *Auth  `json:"auth,omitempty"`
}

// Route represents a single route in the IR.
type Route struct {
	RouteInfo    *RouteInfo         `json:"route"`
	Auth         *Auth              `json:"auth,omitempty"`
	Cache        *Cache             `json:"cache,omitempty"`
	CORS         interface{}        `json:"cors,omitempty"` // *CORS or nil (null for cors(none))
	Input        map[string]*Input  `json:"input,omitempty"`
	Validate     *Validate          `json:"validate,omitempty"`
	TransformIn  map[string]*Transform `json:"transform_in,omitempty"`
	Process      *Process           `json:"process,omitempty"`
	Output       *Output            `json:"output"`
}

// RouteInfo holds the HTTP method and path.
type RouteInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Cache represents HTTP cache directives.
type Cache struct {
	MaxAge       *int        `json:"max_age,omitempty"`
	SMaxAge      *int        `json:"s_maxage,omitempty"`
	Visibility   string      `json:"visibility,omitempty"`
	NoCache      *bool       `json:"no_cache,omitempty"`
	NoStore      *bool       `json:"no_store,omitempty"`
	ETag         interface{} `json:"etag,omitempty"`           // string or *ETagFn
	LastModified string      `json:"last_modified,omitempty"`
	Vary         []string    `json:"vary,omitempty"`
}

// ETagFn represents a function-based etag like hash(user).
type ETagFn struct {
	Fn   string `json:"fn"`
	From string `json:"from"`
}

// CORS represents CORS directives.
type CORS struct {
	Origins       []string `json:"origins,omitempty"`
	Methods       []string `json:"methods,omitempty"`
	Headers       []string `json:"headers,omitempty"`
	ExposeHeaders []string `json:"expose_headers,omitempty"`
	MaxAge        *int     `json:"max_age,omitempty"`
	Credentials   *bool    `json:"credentials,omitempty"`
}

// Auth represents authentication/authorization directives.
type Auth struct {
	Method      string   `json:"method"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Bind        string   `json:"bind,omitempty"`
}

// Input represents an input field extraction.
type Input struct {
	From string `json:"from"`
}

// Validate represents validation rules and error.
type Validate struct {
	Rules map[string]*ValidateRule `json:"rules"`
	Error *ErrorResponse           `json:"error"`
}

// ValidateRule represents a single validation rule.
type ValidateRule struct {
	Type   string `json:"type,omitempty"`
	Min    *int   `json:"min,omitempty"`
	Max    *int   `json:"max,omitempty"`
	Format string `json:"format,omitempty"`
}

// Transform represents a field transformation.
type Transform struct {
	Cast string `json:"cast,omitempty"`
	Fn   string `json:"fn,omitempty"`
	From string `json:"from"`
}

// Process contains the processing steps.
type Process struct {
	Steps []interface{} `json:"steps"` // *PkgStep, *GuardStep, *MatchStep
}

// PkgStep represents a package call step in the process.
type PkgStep struct {
	Bind  string            `json:"bind,omitempty"`
	Use   string            `json:"use"`
	Input map[string]interface{} `json:"input"`
	Error *ErrorResponse    `json:"error,omitempty"`
}

// GuardStep represents a guard step in the process.
type GuardStep struct {
	Guard interface{}    `json:"guard"` // string or map for {"not": "expr"}
	Error *ErrorResponse `json:"error"`
}

// MatchProcessStep represents a match step in the process.
type MatchProcessStep struct {
	Bind  string      `json:"bind,omitempty"`
	Match *MatchBlock `json:"match"`
	Error *ErrorResponse `json:"error,omitempty"`
}

// MatchBlock represents the match block content.
type MatchBlock struct {
	On      string     `json:"on"`
	Arms    []*MatchArm `json:"arms"`
	Default interface{} `json:"default,omitempty"` // *MatchArmAction or *MatchArmError
}

// MatchArm represents a single arm in a match block.
type MatchArm struct {
	Pattern interface{}        `json:"pattern"` // PatternValue, PatternIn, PatternRange, PatternRegex
	Use     string             `json:"use,omitempty"`
	Input   map[string]interface{} `json:"input,omitempty"`
	Error   *ErrorResponse     `json:"error,omitempty"`
	Ref     string             `json:"ref,omitempty"` // variable reference
}

// PatternValue represents a literal match pattern.
type PatternValue struct {
	Value interface{} `json:"value"` // string, int, or bool
}

// PatternIn represents a multi-value match pattern.
type PatternIn struct {
	In []interface{} `json:"in"`
}

// PatternRange represents a range match pattern.
type PatternRange struct {
	Range *RangeValue `json:"range"`
}

// RangeValue holds min/max for a range pattern.
type RangeValue struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// PatternRegex represents a regex match pattern.
type PatternRegex struct {
	Regex string `json:"regex"`
}

// MatchDefaultError is used when the default arm is just an error.
type MatchDefaultError struct {
	Error *ErrorResponse `json:"error"`
}

// Output represents the response output.
type Output struct {
	Status  int               `json:"status"`
	Body    map[string]string `json:"body,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Status int               `json:"status"`
	Body   map[string]string `json:"body,omitempty"`
}
