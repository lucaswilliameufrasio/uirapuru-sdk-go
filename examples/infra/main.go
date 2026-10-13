// Run from your repository:
//
//	go run ./infra > uirapuru.toml     # write the file
//	go run ./infra -check uirapuru.toml # CI: fail when the file is stale
package main

import "github.com/lucaswilliameufrasio/uirapuru-backend/sdk/go/uirapuru"

func main() {
	uirapuru.Main(uirapuru.Manifest{
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
	})
}
