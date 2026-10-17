# HTTP over a Unix domain socket

Researched 2026-10-17 for Taiichi's server/CLI transport. The question: can the Elixir server (Bandit on Thousand Island) serve HTTP + JSON over a Unix domain socket (UDS) instead of a TCP port, so that file permissions control access, and can the likely CLI languages talk to it?

## Answer

**Yes.** Bandit serves HTTP/1.1 and HTTP/2 (h2c) over a UDS with `ip: {:local, path}, port: 0`. Every client asked about can speak HTTP over a UDS, but Zig's standard library is broken (more on that below). The JSON Lines fallback isn't needed.

The caveats are operational, not blockers:

- `port: 0` is mandatory. Leaving `port` out crashes startup.
- Nothing removes the socket file. A stale file from an earlier run makes the next start fail with `:eaddrinuse`.
- The socket file's mode comes from the process umask. Bandit has no permission option.
- `conn.remote_ip` is `{:local, ""}`, which isn't an IP address.
- Clients still need a `Host` header, so they use a placeholder hostname in the URL.
- Zig 0.15–0.17's `std.http.Client.connectUnix` doesn't compile.

## Server side: Bandit and Thousand Island

### Configuration

Bandit passes `ip` through to Thousand Island's transport options, which hands them to `:gen_tcp.listen/2`. Both libraries document the UDS form, and both say `port` must be `0`:

> `{:local, "/path/to/socket"}` for a Unix domain socket. If this option is used, the `port` option *must* be set to `0`

Sources: [Bandit `lib/bandit.ex` L36–38](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/lib/bandit.ex#L36-L38) ([HexDocs](https://hexdocs.pm/bandit/Bandit.html#t:options/0)) and [Thousand Island `transports/tcp.ex` L11–17](https://github.com/mtrudel/thousand_island/blob/1.6.0/lib/thousand_island/transports/tcp.ex#L11-L17) ([HexDocs](https://hexdocs.pm/thousand_island/ThousandIsland.Transports.TCP.html)). OTP describes the address type as [`inet:local_address()`](https://www.erlang.org/doc/apps/kernel/inet.html#t:local_address/0): "In most API functions where you can use this address family the port number must be `0`."

```elixir
children = [
  {Bandit, plug: Taiichi.Router, ip: {:local, "/run/taiichi/taiichi.sock"}, port: 0}
]
```

Here is what happened in a local experiment with Bandit 1.13.1, Thousand Island 1.6.0, Plug 1.20.3, Elixir 1.20.4 and OTP 29 on Linux:

- With `port: 0`, startup logs `Running … with Bandit 1.13.1 at /path/t.sock (http+unix)`. That format string is in [`bandit.ex` L457](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/lib/bandit.ex#L457).
- With `port` left out, Bandit uses its default of 4000 and the listener crashes with `{:function_clause, [{:local_tcp, :getserv, [4000], …}]}`. **Always pass `port: 0`.**
- `ThousandIsland.listener_info/1` returns `{:local, path}`.
- `curl --http2-prior-knowledge --unix-socket …` got `HTTP/2`, so h2c works over the socket too.

### Version history

- **Thousand Island** has documented UDS support since its first commits ([`e97703b`, "Document unix domain socket support", 28 Nov 2019](https://github.com/mtrudel/thousand_island/commit/e97703b)). The package was first published to Hex on 29 Nov 2019.
- **Bandit 1.0.0-pre.10** (28 Jun 2023) fixed "port behaviour when connecting over unix socket (#176)" ([Bandit CHANGELOG](https://github.com/mtrudel/bandit/blob/main/CHANGELOG.md)). So every Bandit 1.x release should work.
- **Thousand Island 1.6.0** (15 Oct 2026) stopped adding its default `nodelay: true` for UDS listeners (#225) ([CHANGELOG](https://github.com/mtrudel/thousand_island/blob/1.6.0/CHANGELOG.md), [`transport.ex` L131–141](https://github.com/mtrudel/thousand_island/blob/1.6.0/lib/thousand_island/transport.ex#L131-L141)). This only matters with `inet_backend: :socket`. The OTP docs say that combination "_will fail_ with `{error, enotsup}`", while the default `inet` backend silently ignores it ([`gen_tcp` docs](https://www.erlang.org/doc/apps/kernel/gen_tcp.html)). I confirmed that 1.6.0 works with `thousand_island_options: [transport_options: [inet_backend: :socket]]`. I did not run 1.5.x to watch it fail.
- Bandit 1.13.x requires `thousand_island ~> 1.6` ([`mix.exs`](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/mix.exs)), so a current Bandit pulls in the fix.

### Socket file permissions

- Bandit, Thousand Island and `gen_tcp` have no option for the socket file's mode, owner or group. I grepped both libraries' `lib/` for `chmod`, `File.rm` and `unlink`, and none of them appear.
- On Linux, the socket file "has all permissions enabled, other than those that are turned off by the process umask(2)". It can be changed afterwards with chown(2) and chmod(2). Connecting "requires write permission on that socket". Creating it needs write and search permission on the directory. See [unix(7)](https://man7.org/linux/man-pages/man7/unix.7.html), "Pathname socket ownership and permissions".
- The same man page warns that POSIX doesn't specify what socket file permissions do, and "on some systems (e.g., older BSDs), the socket permissions are ignored. Portable programs should not rely on this feature for security." FreeBSD does enforce them: "the destination of a connect(2) … must be writable" ([unix(4)](https://man.freebsd.org/cgi/man.cgi?query=unix&sektion=4)). I didn't find a primary source for macOS's behaviour.
- Experiment: under `umask 022` the socket was created `srwxr-xr-x`, so only the owner could connect, since connecting needs write permission. Under `umask 077` it was `srwx------`.

There are three ways to set permissions, most robust first:

1. **Put the socket in a directory only the intended users can reach**, for example `0700`, or `0750` with a group. This is the portable approach the man page recommends, and it closes the window between `bind` and `chmod`. Create the directory, then start Bandit.
2. **Set the umask in the launcher** before the BEAM starts, for example `umask 077` in a release `env.sh` or a systemd `UMask=`. OTP has no API to change the umask at runtime. The setting applies to every file the VM creates.
3. **`File.chmod/2` (and `File.chgrp/2`) after `Bandit.start_link/1` returns.** This works on Linux, but the socket briefly has its umask-derived mode first. It isn't enough alone if the directory is shared.

### Stale socket file on restart

- Closing a socket doesn't remove its file. It "must be deleted by the caller when it is no longer needed (using unlink(2))" ([unix(7)](https://man7.org/linux/man-pages/man7/unix.7.html), NOTES; FreeBSD [unix(4)](https://man.freebsd.org/cgi/man.cgi?query=unix&sektion=4) says the same).
- Experiment: after `GenServer.stop/1` on the Bandit server, the file was still there. The next start failed with `{:error, {:shutdown, {:failed_to_start_child, :listener, :eaddrinuse}}}`. Bandit logged the misleading `… failed, port 0 already in use` ([`bandit.ex` L376](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/lib/bandit.ex#L376)).
- The server has to clean up itself. A safe startup check: `:gen_tcp.connect({:local, path}, 0, [])`.
  - `{:error, :econnrefused}` means a stale file, so delete it.
  - `{:error, :enoent}` means there's nothing to do.
  - A successful connection means another instance is live, so refuse to start.

  I confirmed the first two results by experiment. Deleting the file unconditionally is simpler, but it would silently take the socket path from a second running instance.
- Deleting the file on clean shutdown, for example in a `terminate/2` after Bandit stops, is good hygiene. It doesn't replace the startup check, since a crash or `SIGKILL` skips it.

### Plug behaviour

From Bandit's [`socket_helpers.ex` L9–58](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/lib/bandit/socket_helpers.ex#L9-L58), which maps `{:local, path}` peers to `{{:local, path}, 0}`, and from the experiment:

| Field | Value over UDS | Note |
| --- | --- | --- |
| `conn.remote_ip` | `{:local, ""}` | The client socket is unbound, so the path is empty. This breaks [`Plug.Conn`'s typespec](https://hexdocs.pm/plug/Plug.Conn.html#t:t/0) (`remote_ip: :inet.ip_address()`). Code that formats it with `:inet.ntoa/1` gets `{:error, :einval}`, and code that pattern matches on IP tuples won't match. |
| `Plug.Conn.get_peer_data/1` | `%{address: {:local, ""}, port: 0, ssl_cert: nil}` | Same caveat. |
| `conn.host` | Whatever the client's `Host` header says | Bandit takes host and port from the request target or the `Host` header ([`pipeline.ex` L89–125](https://github.com/mtrudel/bandit/blob/0600a42de8cb4351fab99ca1b35b3afbb72e6e17/lib/bandit/pipeline.ex#L89-L125)). With curl's `http://taiichi/x` it was `"taiichi"`. |
| `conn.port` | `80` | `URI.default_port("http")` when the `Host` header has no port. |
| `conn.scheme` | `:http` | |

The `Host` header is still required on HTTP/1.1. A request with it removed (`-H 'Host:'`) got **400** and logged `Bandit.HTTPError: Unable to obtain host and port: No host header`. HTTP/1.0 without `Host` got 200 with `host: ""`. Every client below sends `Host` from the URL's hostname, so the CLI should use a fixed placeholder like `http://taiichi/…` and the server shouldn't route on host.

Peer credentials (`SO_PEERCRED`, [unix(7)](https://man7.org/linux/man-pages/man7/unix.7.html)) would let the server identify the connecting user. Neither Bandit nor Plug exposes them, so access control rests on file permissions alone. I didn't check whether Thousand Island's raw socket could be queried for them.

### Path length

`sun_path` is 108 bytes on Linux ([unix(7)](https://man7.org/linux/man-pages/man7/unix.7.html)) and 104 on FreeBSD ([unix(4)](https://man.freebsd.org/cgi/man.cgi?query=unix&sektion=4)) and macOS ([xnu `sys/un.h` L79](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/un.h)). OTP notes the limit too ([`inet:local_address()`](https://www.erlang.org/doc/apps/kernel/inet.html#t:local_address/0)). Deep paths, such as those under macOS's `$TMPDIR`, can go over it. Keep the socket path short, for example `$XDG_RUNTIME_DIR/taiichi.sock` or `~/.taiichi/s.sock`.

## Client side

Every client below except Rust was run against the Bandit UDS server above. All returned 200, with the expected `host`.

| Client | Version tested | How | Verified |
| --- | --- | --- | --- |
| curl | 8.22.0 | `curl --unix-socket /path/t.sock http://taiichi/x` | Run |
| Go `net/http` | 1.27.2 | Custom `http.Transport.DialContext` that dials `"unix"` | Run |
| Rust reqwest | (docs only, 0.13.5) | `ClientBuilder::unix_socket(path)`, since 0.12.23 | Docs and changelog |
| Rust hyper | (docs only, 1.x) | `hyper::client::conn::http1::handshake` over a `tokio::net::UnixStream` | Source |
| Zig `std.http.Client` | 0.17.0 | `connectUnix` **does not compile**. Workaround: raw stream | Run (failure and workaround) |
| Bun `fetch` | 1.4.3 | `fetch(url, { unix: path })` | Run |
| Node `http.request` | 26.12.0 | `{ socketPath: path }` | Run |
| Node `fetch` | 26.12.0 | `{ dispatcher: new Agent({ connect: { socketPath } }) }` from the `undici` package | Run |

### curl

`--unix-socket <path>`: "Connect to the server through this Unix domain socket, instead of using the network." Added in 7.40.0 ([curl docs source](https://github.com/curl/curl/blob/master/docs/cmdline-opts/unix-socket.md), [manpage](https://curl.se/docs/manpage.html#--unix-socket)). Linux also has `--abstract-unix-socket`.

### Go (`net/http`)

`net/http` has no UDS URL scheme. `Transport.DialContext` "specifies the dial function for creating unencrypted TCP connections", and you can override it to ignore the address and dial the socket instead ([`transport.go`](https://github.com/golang/go/blob/master/src/net/http/transport.go), [pkg.go.dev](https://pkg.go.dev/net/http#Transport)). Run on Go 1.27.2:

```go
client := &http.Client{Transport: &http.Transport{
	DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", sock)
	},
}}
resp, err := client.Get("http://taiichi/go")
```

Third-party `http+unix` URL helpers exist, such as [`tv42/httpunix`](https://pkg.go.dev/github.com/tv42/httpunix), but aren't needed.

### Rust (reqwest, hyper)

- **reqwest**: `ClientBuilder::unix_socket(path)`: "Set that all connections will use this Unix socket … DNS resolution will not be done on the domain name." It's available on Unix only and isn't compatible with the TCP or proxy options ([docs.rs, reqwest 0.13.5](https://docs.rs/reqwest/latest/reqwest/struct.ClientBuilder.html#method.unix_socket)). It was added in v0.12.23 ([CHANGELOG](https://github.com/seanmonstar/reqwest/blob/master/CHANGELOG.md)). I didn't compile or run it.
- **hyper 1.x**: The low-level client is transport-agnostic. [`client::conn::http1::handshake<T, B>(io: T)`](https://docs.rs/hyper/latest/hyper/client/conn/http1/fn.handshake.html) accepts any `T: Read + Write + Unpin` ([source](https://github.com/hyperium/hyper/blob/master/src/client/conn/http1.rs)), so a `tokio::net::UnixStream` wrapped in `hyper_util::rt::TokioIo` works. The third-party [`hyperlocal`](https://crates.io/crates/hyperlocal) crate packages this, but I didn't check its current maintenance. I didn't compile or run any of this.

### Zig (`std.http.Client`)

`std.http.Client.connectUnix(client, path)` exists, and `request(…, .{ .connection = conn })` would accept the result. But its body wasn't updated when the client was rewritten. In 0.17.0, the latest release ([`Client.zig` L1485–1519](https://codeberg.org/ziglang/zig/src/tag/0.17.0/lib/std/http/Client.zig#L1485-L1519)), it still refers to `ConnectionPool.Node`, `.data`, `Io.net.connectUnixSocket` and a one-argument `addUsed`, and none of those exist anymore. Zig only analyzes functions that get called, so the standard library builds, but any program that calls it fails:

```
lib/std/http/Client.zig:1485:58: error: root source file struct 'posix' has no member named 'SocketError'
```

That's what I got compiling a caller with Zig 0.17.0. Upstream tracks it as [ziglang/zig#31135](https://codeberg.org/ziglang/zig/issues/31135), originally titled "std.http.Client.connectUnix fails to compile in Zig 0.15.x and 0.16.0-dev". The fix, [PR #31768](https://codeberg.org/ziglang/zig/pulls/31768), was still open on 2026-10-17. From reading the source, 0.14 still had the matching `ConnectionPool.Node` type, so the breakage likely began with 0.15's rewrite.

`Connection.Plain.create` is private, so you can't build a pooled connection yourself. What does work in 0.17.0 (compiled and run) is a raw stream from [`std.Io.net.UnixAddress`](https://codeberg.org/ziglang/zig/src/tag/0.17.0/lib/std/Io/net.zig) with the HTTP/1.1 request written by hand:

```zig
const addr = try std.Io.net.UnixAddress.init(path);
const stream = try addr.connect(io);
// stream.writer(io, &buf).interface.writeAll("GET /x HTTP/1.1\r\nHost: taiichi\r\n…")
```

That leaves response parsing (status line, headers, chunked or content-length body) to the CLI. A Zig CLI should vendor a small HTTP/1.1 client, wait for the upstream fix, or treat this as a point against Zig.

### TypeScript on Bun

`fetch` takes a non-standard `unix` option: "To fetch a URL using a Unix domain socket, use the `unix: string` option" ([Bun fetch docs](https://bun.com/docs/api/fetch)). Run on Bun 1.4.3 with a JSON `POST`:

```ts
await fetch("http://taiichi/x", { unix: sock, method: "POST", body: JSON.stringify(payload) });
```

The docs also say a `unix` request "never uses the proxy environment".

### TypeScript on Node

- **`node:http`**: `http.request` takes `socketPath`, a "Unix domain socket. Cannot be used if one of `host` or `port` is specified" ([Node 26 `http` docs](https://nodejs.org/api/http.html#httprequestoptions-callback)). Built in, and it ran.
- **Global `fetch`** has no UDS option of its own. Node accepts a custom undici `dispatcher` ([Node globals: custom dispatcher](https://nodejs.org/api/globals.html#custom-dispatcher)), and undici's `connect.socketPath` is "an IPC endpoint, either a Unix domain socket or a Windows named pipe" ([undici `Client` docs](https://github.com/nodejs/undici/blob/main/docs/docs/api/Client.md)). This needs the `undici` npm package, because Node doesn't expose `Agent`. It ran on Node 26.12.0 with bundled undici 8.11.2 and `undici@8.11.2` installed:

```js
import { Agent } from "undici";
await fetch("http://taiichi/x", { dispatcher: new Agent({ connect: { socketPath: sock } }) });
```

## Implications for Taiichi

- HTTP over a UDS is workable. The JSON Lines fallback isn't needed.
- Pick a socket path under a private directory, create the directory with the intended mode first, and keep the path under 104 bytes.
- On boot, probe the path: delete the file if the connection is refused, and abort if the connection succeeds. Remove the file on clean shutdown.
- Treat `remote_ip` as meaningless. Don't use IP-based plugs, and don't log it with `:inet.ntoa/1`.
- Give the CLI a fixed placeholder host (`http://taiichi/…`), and don't route on `conn.host`.
- Go, Rust (reqwest), Bun and Node have reasonable UDS support. Zig's standard library doesn't today.

## Experiment log

Throwaway, not committed: a `Mix.install([{:bandit, "~> 1.13"}])` script with a plug that echoes `remote_ip`, `host`, `port`, `scheme`, peer data, headers and protocol. It was driven by curl 8.22.0, a Go 1.27.2 program, Node 26.12.0, Bun 1.4.3 and two Zig 0.17.0 programs, on Linux aarch64 (musl). Each run tested the socket mode under umask 022 and 077, restarting over a stale file, omitting `port`, `inet_backend: :socket`, a missing `Host` header and h2c.

## Not confirmed

- Rust reqwest and hyper were checked against docs and source, not run.
- The `nodelay` failure under `inet_backend: :socket` with Thousand Island < 1.6.0 comes from the OTP docs and the changelog. I didn't reproduce it.
- macOS socket permission enforcement: I found no Apple primary source. FreeBSD enforces it, and Linux's man page warns that older BSDs don't.
- Permission denial wasn't exercised as a non-root user, since the experiments ran as root, which bypasses file permissions. The mode bits were observed. The enforcement claim rests on unix(7).
- WebSocket and streaming (chunked or SSE) responses over the UDS weren't tested. They use the same byte stream as plain HTTP/1.1, so I expect no difference.
- Whether Thousand Island's socket can be queried for `SO_PEERCRED`.
