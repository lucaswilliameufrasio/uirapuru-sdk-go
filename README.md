# Uirapuru Go SDK

Describe a project's infrastructure in Go and write it as `uirapuru.toml`.
The hub **never runs this code**: it only reads the TOML the program prints
(design: `docs/iac-manifest.md`). The SDK has no dependencies and cannot carry a
secret value (`Secure()` is only a name).

```go
package main

import "github.com/lucaswilliameufrasio/uirapuru-backend/sdk/go/uirapuru"

func main() {
	uirapuru.Main(uirapuru.Manifest{ /* see examples/infra/main.go */ })
}
```

```
go run ./infra > uirapuru.toml            # write the file
go run ./infra -check uirapuru.toml       # CI: fail when the committed file is stale
uirapuru-validate uirapuru.toml           # CI: the hub's own rules, offline
uirapuru-validate --target staging uirapuru.toml
```

`uirapuru-validate` is built from this repository (`cargo install --git
https://github.com/lucaswilliameufrasio/uirapuru-backend --bin
uirapuru-validate`). It uses the hub's parser and field rules, so what it
accepts the hub accepts; only what depends on the hub's data (clusters,
hosts, conflicts) is left to the plan. `docs/examples/manifest-validate.yml`
is a CI job doing both checks.

Tests: `go test ./...` (set `UIRAPURU_VALIDATE=path/to/uirapuru-validate` to also
run the output through the hub's validator). `testdata/full.toml` is read by the
hub's own test `tests/sdk_golden.rs`, so the two sides cannot drift apart.
