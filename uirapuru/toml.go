package uirapuru

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// TOML renders the manifest. The output is deterministic (maps are sorted), so
// regenerating an unchanged description never produces a diff.
func (m Manifest) TOML() (string, error) {
	var w writer
	w.line("version = %d", Version)

	if m.Options.Prune || m.Options.PruneResources {
		w.blank()
		w.line("[options]")
		if m.Options.Prune {
			w.line("prune = true")
		}
		if m.Options.PruneResources {
			w.line("prune_resources = true")
		}
	}
	for _, name := range sortedKeys(m.Targets) {
		t := m.Targets[name]
		w.blank()
		w.line("[target.%s]", key(name))
		w.str("cluster", t.Cluster)
		w.str("gateway", t.Gateway)
	}
	for _, s := range m.Services {
		if err := writeService(&w, s); err != nil {
			return "", err
		}
	}
	for _, r := range m.Resources {
		if err := writeResource(&w, r); err != nil {
			return "", err
		}
	}
	if w.err != nil {
		return "", w.err
	}
	return w.b.String(), nil
}

func writeService(w *writer, s Service) error {
	if s.Name == "" {
		return errors.New("a service has no name")
	}
	w.blank()
	w.line("[[service]]")
	w.str("name", s.Name)
	w.str("compose", s.Compose)
	w.str("image", s.Image)
	w.str("environment", s.Environment)
	w.uint("replicas", s.Replicas)
	if s.Adopt {
		w.line("adopt = true")
	}
	prefix := "service"
	if err := writeTriggers(w, prefix, s.Triggers); err != nil {
		return fmt.Errorf("service %q: %w", s.Name, err)
	}
	writeRoutes(w, prefix, s.Routes)
	writeEnv(w, prefix, s.Env)
	for _, name := range sortedKeys(s.Targets) {
		o := s.Targets[name]
		p := prefix + ".target." + key(name)
		w.blank()
		w.line("[%s]", p)
		w.str("environment", o.Environment)
		w.uint("replicas", o.Replicas)
		if o.Triggers != nil && len(o.Triggers) == 0 {
			w.line("trigger = []")
		}
		if o.Routes != nil && len(o.Routes) == 0 {
			w.line("route = []")
		}
		if err := writeTriggers(w, p, o.Triggers); err != nil {
			return fmt.Errorf("service %q target %q: %w", s.Name, name, err)
		}
		writeRoutes(w, p, o.Routes)
		writeEnv(w, p, o.Env)
	}
	return nil
}

func writeTriggers(w *writer, prefix string, triggers []Trigger) error {
	for _, t := range triggers {
		if t.Kind == "" {
			return errors.New("a trigger has no kind")
		}
		w.blank()
		w.line("[[%s.trigger]]", prefix)
		w.str("kind", t.Kind)
		if t.Pattern != "" {
			w.str("pattern", t.Pattern)
		}
	}
	return nil
}

func writeRoutes(w *writer, prefix string, routes []Route) {
	for _, r := range routes {
		w.blank()
		w.line("[[%s.route]]", prefix)
		w.str("host", r.Host)
		w.str("path_prefix", r.PathPrefix)
		w.line("port = %d", r.Port)
	}
}

func writeEnv(w *writer, prefix string, env map[string]Env) {
	if len(env) == 0 {
		return
	}
	w.blank()
	w.line("[%s.env]", prefix)
	for _, name := range sortedKeys(env) {
		e := env[name]
		if e.kind == "" {
			w.fail(fmt.Errorf("variable %q is neither Plain nor Secure", name))
			continue
		}
		if e.kind == "secure" {
			w.line("%s = { kind = \"secure\" }", key(name))
		} else {
			w.line("%s = { kind = \"plain\", value = %s }", key(name), quote(e.value))
		}
	}
}

func writeResource(w *writer, r Resource) error {
	if r.Name == "" {
		return errors.New("a resource has no name")
	}
	w.blank()
	w.line("[[resource]]")
	w.str("name", r.Name)
	w.str("template", r.Template)
	w.str("version", r.Version)
	w.str("mode", r.Mode)
	w.str("server", r.Server)
	w.str("service", r.Service)
	w.str("env_name", r.EnvName)
	if r.Adopt {
		w.line("adopt = true")
	}
	if len(r.Config) > 0 {
		return writeTable(w, "resource.config", r.Config)
	}
	return nil
}

// writeTable writes a config map: scalars first, then nested tables.
func writeTable(w *writer, path string, table map[string]any) error {
	w.blank()
	w.line("[%s]", path)
	var nested []string
	for _, k := range sortedKeys(table) {
		if _, ok := table[k].(map[string]any); ok {
			nested = append(nested, k)
			continue
		}
		v, err := scalar(table[k])
		if err != nil {
			return fmt.Errorf("%s.%s: %w", path, k, err)
		}
		w.line("%s = %s", key(k), v)
	}
	for _, k := range nested {
		if err := writeTable(w, path+"."+key(k), table[k].(map[string]any)); err != nil {
			return err
		}
	}
	return nil
}

func scalar(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return quote(x), nil
	case bool:
		return strconv.FormatBool(x), nil
	case int:
		return strconv.Itoa(x), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case uint32:
		return strconv.FormatUint(uint64(x), 10), nil
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64), nil
	case []string:
		items := make([]string, len(x))
		for i, s := range x {
			items[i] = quote(s)
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	case []any:
		items := make([]string, len(x))
		for i, e := range x {
			s, err := scalar(e)
			if err != nil {
				return "", err
			}
			items[i] = s
		}
		return "[" + strings.Join(items, ", ") + "]", nil
	}
	return "", fmt.Errorf("unsupported config value of type %T", v)
}

type writer struct {
	b   strings.Builder
	err error
}

func (w *writer) line(format string, args ...any) {
	w.b.WriteString(fmt.Sprintf(format, args...))
	w.b.WriteByte('\n')
}

// blank separates tables, but never starts the file with one.
func (w *writer) blank() {
	if w.b.Len() > 0 {
		w.b.WriteByte('\n')
	}
}

func (w *writer) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *writer) str(name, value string) {
	if value != "" {
		w.line("%s = %s", name, quote(value))
	}
}

func (w *writer) uint(name string, value *uint32) {
	if value != nil {
		w.line("%s = %d", name, *value)
	}
}

// key writes a table or variable name, quoted when it is not a bare TOML key.
func key(name string) string {
	if name != "" && strings.IndexFunc(name, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-')
	}) < 0 {
		return name
	}
	return quote(name)
}

// quote writes a TOML basic string.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Check compares what the manifest renders with the file at path, for CI: it
// fails when the committed uirapuru.toml is not what the Go code describes.
func (m Manifest) Check(path string) error {
	want, err := m.TOML()
	if err != nil {
		return err
	}
	have, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(have) != want {
		return fmt.Errorf("%s is out of date: regenerate it (go run ./infra > %s)", path, path)
	}
	return nil
}
