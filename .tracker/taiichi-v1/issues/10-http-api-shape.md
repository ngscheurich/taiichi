# HTTP API shape

Type: grilling
Status: open
Blocked by: 04, 06, 14

## Question

What HTTP API does the server expose to the CLI? Resources, endpoints, request and response bodies, error format, and versioning, derived from the commands in [CLI commands](06-cli-commands.md) and the transport confirmed by [HTTP over a Unix socket](04-http-over-unix-socket.md).

From [Revisit the transport and protocol](14-revisit-transport-and-protocol.md): HTTP + JSON over a Unix socket is confirmed. The API doesn't have to be REST: CLI commands are actions, so RPC-style endpoints (such as `POST /submit-output`, JSON in and out) are a valid shape to weigh against resources and verbs.
