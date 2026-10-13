package uirapuru_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucaswilliameufrasio/uirapuru-backend/sdk/go/uirapuru"
)

func full() uirapuru.Manifest {
	return uirapuru.Manifest{
		Targets: map[string]uirapuru.Target{
			"default": {Cluster: "vps-1", Gateway: "edge"},
			"staging": {Cluster: "vps-staging", Gateway: "edge-staging"},
		},
		Services: []uirapuru.Service{{
			Name:     "api",
			Image:    "ghcr.io/acme/api",
			Replicas: uirapuru.Uint(2),
			Triggers: []uirapuru.Trigger{{Kind: "push", Pattern: "main"}},
			Routes:   []uirapuru.Route{{Host: "api.example.com", Port: 8080}},
			Env: map[string]uirapuru.Env{
				"LOG_LEVEL":    uirapuru.Plain("info"),
				"DATABASE_URL": uirapuru.Secure(),
			},
			Targets: map[string]uirapuru.ServiceOverride{
				"staging": {
					Replicas: uirapuru.Uint(1),
					Routes:   []uirapuru.Route{{Host: "api.staging.example.com", Port: 8080}},
				},
			},
		}},
		Resources: []uirapuru.Resource{
			{Name: "db", Template: "postgres", Mode: "dedicated", Server: "db-host", Version: "17"},
			{Name: "cache", Template: "valkey", Mode: "shared", Service: "api", EnvName: "VALKEY_URL"},
		},
	}
}

// The same description in testdata/full.toml is read by the hub's own test
// (tests/sdk_golden.rs), so both sides agree on the format.
func TestRendersTheGoldenFile(t *testing.T) {
	want, err := os.ReadFile("../testdata/full.toml")
	if err != nil {
		t.Fatal(err)
	}
	got, err := full().TOML()
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("output differs from testdata/full.toml:\n%s", got)
	}
}

func TestOutputIsDeterministic(t *testing.T) {
	first, _ := full().TOML()
	for i := 0; i < 20; i++ {
		again, _ := full().TOML()
		if again != first {
			t.Fatal("two renders of the same manifest differ")
		}
	}
}

func TestSecureVariablesNeverCarryAValue(t *testing.T) {
	m := uirapuru.Manifest{Services: []uirapuru.Service{{
		Name: "api", Env: map[string]uirapuru.Env{"TOKEN": uirapuru.Secure()},
	}}}
	out, err := m.TOML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `TOKEN = { kind = "secure" }`) || strings.Contains(out, "value") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestQuotesWhatNeedsQuoting(t *testing.T) {
	m := uirapuru.Manifest{Services: []uirapuru.Service{{
		Name: "api",
		Env:  map[string]uirapuru.Env{"MSG": uirapuru.Plain("say \"hi\"\\\nbye\t\x01"), "odd.name": uirapuru.Plain("x")},
	}}}
	out, _ := m.TOML()
	for _, want := range []string{`value = "say \"hi\"\\\nbye\t\u0001"`, `"odd.name" = {`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in:\n%s", want, out)
		}
	}
}

func TestAnEmptyOverrideListReplacesWithNone(t *testing.T) {
	m := uirapuru.Manifest{Services: []uirapuru.Service{{
		Name:    "api",
		Routes:  []uirapuru.Route{{Host: "a.example.com", Port: 80}},
		Targets: map[string]uirapuru.ServiceOverride{"staging": {Routes: []uirapuru.Route{}}},
	}}}
	out, _ := m.TOML()
	if !strings.Contains(out, "[service.target.staging]\nroute = []") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestResourceConfigNests(t *testing.T) {
	m := uirapuru.Manifest{Resources: []uirapuru.Resource{{
		Name: "q", Template: "nats", Mode: "dedicated", Server: "s",
		Config: map[string]any{"jetstream": true, "limits": map[string]any{"max_mb": 512, "tags": []string{"a", "b"}}},
	}}}
	out, err := m.TOML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[resource.config]\njetstream = true") || !strings.Contains(out, "[resource.config.limits]\nmax_mb = 512\ntags = [\"a\", \"b\"]") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestMistakesAreErrors(t *testing.T) {
	cases := map[string]uirapuru.Manifest{
		"no service name":  {Services: []uirapuru.Service{{}}},
		"no trigger kind":  {Services: []uirapuru.Service{{Name: "a", Triggers: []uirapuru.Trigger{{}}}}},
		"zero-value env":   {Services: []uirapuru.Service{{Name: "a", Env: map[string]uirapuru.Env{"X": {}}}}},
		"no resource name": {Resources: []uirapuru.Resource{{}}},
		"bad config type":  {Resources: []uirapuru.Resource{{Name: "r", Config: map[string]any{"x": struct{}{}}}}},
	}
	for name, m := range cases {
		if _, err := m.TOML(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestCheckDetectsAStaleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "uirapuru.toml")
	out, _ := full().TOML()
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := full().Check(path); err != nil {
		t.Fatalf("fresh file reported stale: %v", err)
	}
	if err := os.WriteFile(path, []byte(out+"# edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := full().Check(path); err == nil || !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("stale file not detected: %v", err)
	}
}

// With UIRAPURU_VALIDATE=/path/to/uirapuru-validate the output also has to pass
// the hub's own validator for every target.
func TestPassesTheHubValidator(t *testing.T) {
	bin := os.Getenv("UIRAPURU_VALIDATE")
	if bin == "" {
		t.Skip("set UIRAPURU_VALIDATE to the uirapuru-validate binary")
	}
	out, _ := full().TOML()
	path := filepath.Join(t.TempDir(), "uirapuru.toml")
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"default", "staging"} {
		if msg, err := exec.Command(bin, "--target", target, path).CombinedOutput(); err != nil {
			t.Errorf("target %s rejected: %v\n%s", target, err, msg)
		}
	}
}
