# rcsb

A command line for rcsb.

`rcsb` is a single pure-Go binary. It reads public rcsb data
over plain HTTPS, shapes it into clean records, and prints output that pipes
into the rest of your tools. No API key, nothing to run alongside it.

The same package is also a [resource-URI driver](#use-it-as-a-resource-uri-driver),
so a host program like [ant](https://github.com/tamnd/ant) can address
rcsb as `rcsb://` URIs.

## Install

```bash
go install github.com/tamnd/rcsb-cli/cmd/rcsb@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/rcsb-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/rcsb:latest --help
```

## Usage

```bash
rcsb page <path>                      # fetch one page as a record
rcsb page <path> -o json              # as JSON, ready for jq
rcsb page <path> --template '{{.Body}}'  # just the readable body text
rcsb links <path>                     # the pages it links to, one per line
rcsb --help                           # the whole command tree
```

Every command shares one output contract: `-o table|json|jsonl|csv|tsv|url|raw`,
`--fields` to pick columns, `--template` for a custom line, and `-n` to limit.
The default adapts to where output goes (a table on a terminal, JSONL in a
pipe), so the same command reads well by hand and parses cleanly downstream.

This is a fresh scaffold. It ships one example resource type, `page`, wired end
to end. Model the real rcsb records in `rcsb/` and declare their
operations in `rcsb/domain.go`; each one becomes a command, an HTTP
route, and an MCP tool at once.

## Serve it

The same operations are available over HTTP and as an MCP tool set for agents,
with no extra code:

```bash
rcsb serve --addr :7777    # GET /v1/page/<path>  returns NDJSON
rcsb mcp                   # speak MCP over stdio
```

## Use it as a resource-URI driver

`rcsb` registers a `rcsb` domain the way a program registers a
database driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/rcsb-cli/rcsb"
```

Then [ant](https://github.com/tamnd/ant) (or any program that links the package)
dereferences `rcsb://` URIs without knowing anything about rcsb:

```bash
ant get rcsb://page/<path>   # fetch the record
ant cat rcsb://page/<path>   # just the body text
ant ls  rcsb://page/<path>   # the pages it links to, each addressable
ant url rcsb://page/<path>   # the live https URL
```

## Development

```
cmd/rcsb/   thin main: hands cli.NewApp to kit.Run
cli/                 assembles the kit App from the rcsb domain
rcsb/                the library: HTTP client, data models, and domain.go (the driver)
docs/                tago documentation site
```

```bash
make build      # ./bin/rcsb
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.0
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
