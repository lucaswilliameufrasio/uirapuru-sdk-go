// Package uirapuru describes a project's infrastructure in Go and writes it as
// a uirapuru.toml. The hub never runs this code: it reads only the TOML the
// program prints, so the file in the repository stays the single contract and
// can be reviewed like any other change. Nothing here can carry a secret
// value: a secure variable is only a name.
//
//	m := uirapuru.Manifest{...}
//	uirapuru.Main(m) // prints the TOML; with -check FILE, fails if FILE differs
package uirapuru

// Version is the manifest format this package writes.
const Version = 1

// Manifest is a whole uirapuru.toml.
type Manifest struct {
	Options   Options
	Targets   map[string]Target
	Services  []Service
	Resources []Resource
}

// Options are the file-wide switches. Prune is refused by the hub today.
type Options struct {
	Prune          bool
	PruneResources bool
}

// Target names the entities that only exist in a given hub.
type Target struct {
	Cluster string
	Gateway string
}

// Service is a deployable unit of the project.
type Service struct {
	Name        string
	Compose     string
	Image       string
	Environment string
	Replicas    *uint32
	Adopt       bool
	Triggers    []Trigger
	Routes      []Route
	Env         map[string]Env
	// Targets change the service for one target only.
	Targets map[string]ServiceOverride
}

// ServiceOverride is what one target changes. A nil slice keeps the service's
// own list; an empty, non-nil slice replaces it with none.
type ServiceOverride struct {
	Environment string
	Replicas    *uint32
	Triggers    []Trigger
	Routes      []Route
	Env         map[string]Env
}

// Trigger says which event deploys the service: kind is push, tag, image or manual.
type Trigger struct {
	Kind    string
	Pattern string
}

// Route publishes the service on a host through the target's gateway.
type Route struct {
	Host       string
	PathPrefix string // "/" when empty
	Port       uint32
}

// Env is a variable: plain with a value, or secure with only its name.
type Env struct {
	kind  string
	value string
}

// Plain is a variable whose value lives in the repository.
func Plain(value string) Env { return Env{kind: "plain", value: value} }

// Secure declares a variable whose value is set in the hub (it goes to Ciphera).
func Secure() Env { return Env{kind: "secure"} }

// Resource is infrastructure the hub provisions for the project.
type Resource struct {
	Name     string
	Template string // postgres, valkey, nats or custom
	Version  string
	Mode     string // dedicated or shared: always explicit
	Server   string // dedicated only
	Service  string // shared only: the service that receives the variable
	EnvName  string // shared only
	Config   map[string]any
	Adopt    bool
}

// Uint returns a pointer for the optional counts (Replicas).
func Uint(n uint32) *uint32 { return &n }
