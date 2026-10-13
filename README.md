# Uirapuru Go SDK

Describe a project's infrastructure in Go and write it as `uirapuru.toml`.
The hub **never runs this code**: it only reads the TOML the program prints
(design: `docs/iac-manifest.md` in
[uirapuru-backend](https://github.com/lucaswilliameufrasio/uirapuru-backend)). The SDK has no dependencies and cannot carry a
secret value (`Secure()` is only a name).

```go
package main

import "github.com/lucaswilliameufrasio/uirapuru-sdk-go/uirapuru"

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

`uirapuru-validate` is built from the hub repository (`cargo install --git
https://github.com/lucaswilliameufrasio/uirapuru-backend --bin
uirapuru-validate`). It uses the hub's parser and field rules, so what it
accepts the hub accepts; only what depends on the hub's data (clusters,
hosts, conflicts) is left to the plan. The hub repository has a CI job doing both
checks (`docs/examples/manifest-validate.yml`).

Install: `go get github.com/lucaswilliameufrasio/uirapuru-sdk-go@latest`.

Tests: `go test ./...` (set `UIRAPURU_VALIDATE=path/to/uirapuru-validate` to also
run the output through the hub's validator). `testdata/full.toml` has a copy in
the hub (`tests/fixtures/sdk_full.toml`, read by its test `tests/sdk_golden.rs`),
and the hub's CI runs this repository's tests against the real validator, so the
two sides cannot drift apart: change the golden file here and there together.

## Layout of the SDKs

Each language has its own repository, `uirapuru-sdk-<language>`
(`uirapuru-sdk-go` here), versioned on its own with plain `vX.Y.Z` tags. An SDK
only renders `uirapuru.toml`; the format and its rules live in the hub.
