# Revisit the transport and protocol

Type: grilling
Status: resolved

## Question

Is HTTP + JSON over a Unix socket still the right way for the CLI and the server to talk? [Chart the v1 scope](01-chart-the-v1-scope.md) chose it over a Unix socket with JSON Lines, and [HTTP over a Unix socket](04-http-over-unix-socket.md) confirmed Bandit and the likely CLI languages support it. The user wants to reopen the choice of transport and protocol before the API and CLI language are designed on top of it.

**Why it's reopened:** the user doesn't fully understand the trade-offs behind the current choice. Start by laying them out plainly before asking for any decision, keeping the two choices separate:

- **Transport** (how bytes travel): a Unix socket versus TCP on localhost.
- **Protocol** (what the bytes mean): HTTP + JSON versus JSON-RPC. The user meant JSON-RPC when they first raised "JSON Lines" as the alternative; the earlier rounds and [Chart the v1 scope](01-chart-the-v1-scope.md) compared against JSON Lines by mistake. JSON Lines matters here only as one possible way of framing JSON-RPC messages on a socket.

For each option, explain what it makes easier or harder against this map's actual needs: a non-Elixir CLI; future AI agents as clients, possibly in sandboxes; what else on the machine can reach the server (other local processes, web pages open in a browser); the fogged "watching a Job live" idea, which needs the server to push updates; and the server-side work the research found (stale socket files, socket permissions). Don't justify options with premises the user hasn't stated.

## Answer

**Keep HTTP + JSON over a Unix socket.** Settled with the user after laying out the trade-offs, with transport and protocol kept separate.

**Transport: a Unix socket, not TCP on localhost.**
- Only the user's own account (and root) can connect, since the socket lives in a `0700` directory. File permissions do this with no code. On TCP, any local user could connect unless the server added a token.
- Browsers can't reach a Unix socket. On TCP, a web page can send requests to localhost, so the server would need `Origin` checks or a token.
- This matters because the server holds the Forgejo token and creates branches and worktrees.
- The extra server work is small and already specified by [HTTP over a Unix socket](04-http-over-unix-socket.md): probe for and clear a stale socket file, and create the private directory.
- Sandboxed AI agents didn't decide it. Either transport needs the sandbox to let it through: the socket's directory shared in, or the host's network reachable.

**Protocol: HTTP + JSON, not JSON-RPC.**
- Bandit and Plug supply routing, parsing, errors and timeouts. JSON-RPC on a raw socket would mean the server writes its own connection handling, framing and dispatch.
- Debugging works with `curl --unix-socket`.
- Two-way push isn't a JSON-RPC advantage. A WebSocket, which Bandit supports, gives HTTP a full two-way connection when one is needed. Server-sent events cover one-way push.
- The one thing that would have tipped the choice to JSON-RPC is choosing Zig for the CLI, because Zig's standard HTTP client can't use a Unix socket.
- On HTTP feeling heavier: it brings more protocol surface to learn and design (verbs, status codes, headers, a placeholder `Host`), but less code to own on both sides. The REST pull is optional: RPC-style endpoints are valid HTTP. [HTTP API shape](10-http-api-shape.md) decides between them.

The earlier rounds compared against JSON Lines by mistake; the user meant JSON-RPC. JSON Lines is relevant only as a way to frame JSON-RPC on a socket, so it drops out with JSON-RPC.

No glossary change and no ADR: no domain terms moved, and the choice isn't surprising enough to need an ADR.
