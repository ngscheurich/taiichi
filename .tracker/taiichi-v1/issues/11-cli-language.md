# CLI language

Type: grilling
Status: open
Blocked by: 04, 06, 14

## Question

Which language is the CLI written in? Weigh how much logic the CLI holds (from [CLI commands](06-cli-commands.md)) and client support for HTTP over a Unix socket (from [HTTP over a Unix socket](04-http-over-unix-socket.md)). Single-binary distribution is not a major factor.

From [HTTP over a Unix socket](04-http-over-unix-socket.md): curl, Go, Bun and Node were confirmed working, Rust is supported per docs, and Zig's standard HTTP client can't use a Unix socket today.
