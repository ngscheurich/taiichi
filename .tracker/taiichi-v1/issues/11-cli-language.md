# CLI language

Type: grilling
Status: resolved
Blocked by: 04, 06, 14

## Question

Which language is the CLI written in? Weigh how much logic the CLI holds (from [CLI commands](06-cli-commands.md)) and client support for HTTP over a Unix socket (from [HTTP over a Unix socket](04-http-over-unix-socket.md)). Single-binary distribution is not a major factor.

From [HTTP over a Unix socket](04-http-over-unix-socket.md): curl, Go, Bun and Node were confirmed working, Rust is supported per docs, and Zig's standard HTTP client can't use a Unix socket today.

From [Revisit the transport and protocol](14-revisit-transport-and-protocol.md): HTTP + JSON over a Unix socket is confirmed. Choosing Zig would mean a hand-written HTTP/1.1 client.

## Comments

**From [`taiichi init` and configuration](09-init-and-configuration.md):** the CLI never parses `taiichi.toml` or `server.toml`. It runs `git rev-parse` to find the Repo, works out the socket path (`TAIICHI_SOCKET`, else the XDG default), and sends requests.

**From [HTTP API shape](10-http-api-shape.md):** what the CLI holds. Each command runs `git rev-parse` for the common dir (and `--show-toplevel` plus a `locate` call when the Job argument is left out), then `POST`s JSON to `http://taiichi/v1/<method>` over the socket. The server returns structured data, so the CLI lays out tables and history, prints a Stage Input's parts in order, and prints a document's `meta` as `key: value` lines. It also works out "what's next" from `position` and `awaiting`, runs `$EDITOR` with the scissors line, asks for confirmations, and makes `init`'s second call with `create_labels: true`. Messages come from the server, and the CLI branches only on error `code`s.

## Answer

Settled with the user, after research, two working prototypes and a restyle pass to separate the builders' choices from the ecosystems' limits.

**The CLI is written in Go, with Charm.**
- **Stack:** Cobra for commands and completions, Fang to style Cobra's help and errors, Huh for prompts, Lip Gloss for tables and styling, Glamour for Markdown, and Bubble Tea v2 for the post-v1 TUI.
- **Elixir was ruled out by the user:** they want an established CLI framework with great UX, and Elixir has no good TUI library for the planned post-v1 TUI.
- **The runner-up was TypeScript on Bun:** Bombshell's Router, Clack and Tab, `util.styleText` and `Bun.markdown`, with OpenTUI as the user's pick for its TUI.
- **Why Go won:** the user found the Go prototype more polished. After the Bun prototype was restyled to match Go's choices (palette, bordered tables, light and dark Markdown, a paged wizard), our own output matched. The gap that remained sits in what each ecosystem provides:
  - Clack's colours are hard-coded.
  - Router's help is plain, while Fang styles Cobra's.
  - `Bun.markdown` is thinner than Glamour: light mode only changes inline code, it highlights only JS/TS, and it doesn't downsample.
  - OpenTUI has no form library, so a paged wizard meant about 200 lines of our own form code.
- **What TypeScript would have given:** two languages instead of three, and one API client shared with the future harness. Neither gain arrives until the harness exists, after v1.
- **A checkpoint before the TUI:** switching is cheap until the TUI exists, because the CLI is a thin client. A switch rewrites only its UI code and an 11-method API client. The TUI and the wizard don't carry over. Tracked as [Revisit the CLI language before the TUI](../../taiichi-post-v1/issues/03-revisit-cli-language-before-tui.md).

**Repo layout:** `server/` (the Mix app moves here) and `cli/` sit side by side at the top level. The root holds what both share: `.tracker/`, `docs/`, `GLOSSARY.md` and `mise.toml` with both toolchains. One repo lets one commit change an API method and the CLI call that uses it.

**The future harness** runs as its own long-running TypeScript process hosting pi durable (from `earendil-works/pi`), as a client of the server's API, like the CLI. pi durable is a TypeScript-only, in-process library that needs one long-lived process to own its store. Hosting it in the TUI would tie the agent's lifetime to an open terminal and need one TUI per running Job. This is input to the map's executor-seam fog item, not a settlement of it.

**`cat` grammar changed:** `taiichi cat <stage> [job] [-n <attempt>]`. The Job is the last positional on every command; for every other Job command it's the only positional, so nothing else changes. The old `cat [job] <stage> [<n>]` couldn't be parsed by left-to-right parsers such as Router. The new form parses in both Cobra and Router.

**For whoever builds it** (from the prototypes):
- **Prompts:** check for a terminal ourselves before every prompt and fail with a message naming `--yes`. Huh opens `/dev/tty` even when stdin is redirected.
- **Fang's background query:** it stalls about 4 s on a terminal that never answers. Use a short timeout, or set the colour scheme explicitly.
- **Glamour:**
  - Detect the background ourselves, since v2 removed auto-detection.
  - Print Markdown raw when stdout isn't a terminal: Glamour emits colour and padding even through `lipgloss.Print`.
- **Huh's standalone forms:** they never query the background, so themes render light (huh#834). A 6-line fix is in the prototype.
- **The post-v1 wizard:** build it from Huh's fields inside our own Bubble Tea model. Huh can only hide whole groups, can't run a check between groups, and has an open multi-select height bug (huh#831). It has had no release since 2026-03-10.
- **The editor:** resolve it with `git var GIT_EDITOR`, and hand it the terminal with `tea.ExecProcess`.

**Assets**
- Research, pi and pi durable: `docs/research/pi-integration.md` on branch `research/pi-integration` (`7771aa1`).
- Research, Bombshell against Charm: `docs/research/bombshell-vs-charm.md` on branch `research/bombshell-vs-charm` (`34f9df3`).
- Prototypes: branch `prototype/cli-stacks` (`13eb68f`). The spec and fake data are in `cli/PROTOTYPES.md` and `cli/prototype-fixtures/`. The two builds are `cli/prototype-go/` and `cli/prototype-bun/`, each with a `NOTES.md`.
- None of these branches is pushed; the sandbox has no `ssh`.

No glossary change and no ADR: no domain terms moved, and the choice is cheap to reverse until the TUI exists.
