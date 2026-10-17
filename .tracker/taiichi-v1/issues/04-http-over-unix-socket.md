# HTTP over a Unix socket: Bandit and client support

Type: research
Status: resolved

## Question

The plan is for the Elixir server to serve HTTP + JSON over a Unix domain socket. Can it?

- Can Bandit (through Thousand Island) listen on a Unix socket? How is it configured, and what are the caveats (socket file permissions, removing a stale socket file on restart, Plug behaviour such as `conn.remote_ip`)?
- Can common clients speak HTTP over a Unix socket: `curl --unix-socket`, and the standard HTTP clients of likely CLI languages (Go, Rust, Zig, TypeScript on Bun or Node)?

Primary sources: Bandit and Thousand Island source and docs, and each client's official docs.

## Answer

**Yes.** Bandit serves HTTP/1.1 and h2c over a Unix socket with `ip: {:local, path}, port: 0` (confirmed locally with Bandit 1.13.1, Thousand Island 1.6.0, OTP 29). The JSON Lines fallback isn't needed.

Caveats the server must handle:
- **`port: 0` is required;** omitting it crashes startup.
- **Stale socket file:** nothing removes it on shutdown, and the next start fails with `:eaddrinuse` (logged misleadingly as "port 0 already in use"). Probe at startup: connection refused means stale (delete it), no file means clear, a successful connection means another server is running (abort).
- **Permissions:** the socket's mode comes from the process umask; no library sets it. Put the socket in a `0700` directory rather than relying on the socket's own mode.
- **Plug fields:** `conn.remote_ip` is `{:local, ""}` (outside Plug's declared type); `conn.host` comes from the `Host` header, which HTTP/1.1 requires, so clients use a placeholder URL such as `http://taiichi/...`.

Clients: confirmed working with curl (`--unix-socket`), Go (custom `DialContext`), Bun (`fetch` with `unix`), and Node (`http.request` with `socketPath`, or `fetch` with an undici agent). Rust's reqwest (0.12.23+) and hyper support it per docs and source, not run. **Zig's `std.http.Client` can't do it** in 0.15–0.17 (ziglang/zig#31135, fix still open); only a hand-written HTTP/1.1 client over a raw socket works.

Findings: `docs/research/http-over-unix-socket.md` on branch `research/http-over-unix-socket` (commit `f82044b`).
