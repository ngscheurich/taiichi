# Revisit the transport and protocol

Type: grilling
Status: open

## Question

Is HTTP + JSON over a Unix socket still the right way for the CLI and the server to talk? [Chart the v1 scope](01-chart-the-v1-scope.md) chose it over a Unix socket with JSON Lines, and [HTTP over a Unix socket](04-http-over-unix-socket.md) confirmed Bandit and the likely CLI languages support it. The user wants to reopen the choice of transport and protocol before the API and CLI language are designed on top of it.

**Why it's reopened:** the user doesn't fully understand the trade-offs behind the current choice. Start by laying them out plainly before asking for any decision, keeping the two choices separate:

- **Transport** (how bytes travel): a Unix socket versus TCP on localhost.
- **Protocol** (what the bytes mean): HTTP + JSON versus JSON-RPC. The user meant JSON-RPC when they first raised "JSON Lines" as the alternative; the earlier rounds and [Chart the v1 scope](01-chart-the-v1-scope.md) compared against JSON Lines by mistake. JSON Lines matters here only as one possible way of framing JSON-RPC messages on a socket.

For each option, explain what it makes easier or harder against this map's actual needs: a non-Elixir CLI; future AI agents as clients, possibly in sandboxes; what else on the machine can reach the server (other local processes, web pages open in a browser); the fogged "watching a Job live" idea, which needs the server to push updates; and the server-side work the research found (stale socket files, socket permissions). Don't justify options with premises the user hasn't stated.
