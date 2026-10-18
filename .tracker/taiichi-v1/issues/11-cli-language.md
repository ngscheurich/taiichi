# CLI language

Type: grilling
Status: open
Blocked by: 04, 06, 14

## Question

Which language is the CLI written in? Weigh how much logic the CLI holds (from [CLI commands](06-cli-commands.md)) and client support for HTTP over a Unix socket (from [HTTP over a Unix socket](04-http-over-unix-socket.md)). Single-binary distribution is not a major factor.

From [HTTP over a Unix socket](04-http-over-unix-socket.md): curl, Go, Bun and Node were confirmed working, Rust is supported per docs, and Zig's standard HTTP client can't use a Unix socket today.

From [Revisit the transport and protocol](14-revisit-transport-and-protocol.md): HTTP + JSON over a Unix socket is confirmed. Choosing Zig would mean a hand-written HTTP/1.1 client.

## Comments

**From [`taiichi init` and configuration](09-init-and-configuration.md):** the CLI never parses `taiichi.toml` or `server.toml`. It runs `git rev-parse` to find the Repo, works out the socket path (`TAIICHI_SOCKET`, else the XDG default), and sends requests.

**From [HTTP API shape](10-http-api-shape.md):** what the CLI holds. Each command runs `git rev-parse` for the common dir (and `--show-toplevel` plus a `locate` call when the Job argument is left out), then `POST`s JSON to `http://taiichi/v1/<method>` over the socket. The server returns structured data, so the CLI lays out tables and history, prints a Stage Input's parts in order, and prints a document's `meta` as `key: value` lines. It also works out "what's next" from `position` and `awaiting`, runs `$EDITOR` with the scissors line, asks for confirmations, and makes `init`'s second call with `create_labels: true`. Messages come from the server, and the CLI branches only on error `code`s.
