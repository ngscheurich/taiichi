# Taiichi

A software factory that pulls Tickets from a Tracker and tracks each one as it moves through a fixed Pipeline of Stages. [GLOSSARY.md](GLOSSARY.md) defines these terms.

Taiichi has two programs, side by side in this repository:

- **`server/`**: the server, a Mix app in Elixir. It holds every rule and is the only program that writes Job data.
- **`cli/`**: the CLI, `taiichi`, a Go module. It sends requests to the server and prints the answers.

## Setup

[mise](https://mise.jdx.dev) installs the Elixir, Erlang and Go versions pinned in `mise.toml`:

```sh
mise install
```

## The server

Run these from `server/`:

| What | Command |
|---|---|
| Fetch dependencies | `mix deps.get` |
| Build | `mix compile --warnings-as-errors` |
| Test | `mix test` |

## The CLI

Run these from `cli/`:

| What | Command |
|---|---|
| Build | `go build -o bin/taiichi ./cmd/taiichi` |
| Test | `go test ./...` |

The build writes the binary to `cli/bin/taiichi`.
