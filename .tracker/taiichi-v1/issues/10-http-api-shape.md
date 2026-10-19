# HTTP API shape

Type: grilling
Status: resolved
Blocked by: 04, 06, 14

## Question

What HTTP API does the server expose to the CLI? Resources, endpoints, request and response bodies, error format, and versioning, derived from the commands in [CLI commands](06-cli-commands.md) and the transport confirmed by [HTTP over a Unix socket](04-http-over-unix-socket.md).

From [Revisit the transport and protocol](14-revisit-transport-and-protocol.md): HTTP + JSON over a Unix socket is confirmed. The API doesn't have to be REST: CLI commands are actions, so RPC-style endpoints (such as `POST /submit-output`, JSON in and out) are a valid shape to weigh against resources and verbs.

## Answer

Settled with the user. Glossary: **Position** added (where a Job stands in the Pipeline: a Stage at an Attempt, or Done or Abandoned), with "state" and "status" to avoid.

**Style: plain RPC over HTTP**
- Every call is `POST /v1/<method>`, JSON in and out (`Content-Type: application/json`), with the placeholder host `http://taiichi` from [HTTP over a Unix socket](04-http-over-unix-socket.md).
- All RPC, reads included. A hybrid of `GET` reads and `POST` actions was rejected: it means two mental models, and since no scripts exist yet, their authors read the docs either way.
- JSON-RPC 2.0 was weighed and rejected. It runs on top of HTTP rather than replacing it, so clients would handle two error layers (Plug's HTTP errors and JSON-RPC errors inside a 200). It defines no HTTP binding, so we'd still invent the HTTP side. Its error codes are integers. Conformance brings notifications (a request without an `id` gets no reply) and batches, which we'd never use. Its gains, a spec name and messages ready for a WebSocket, didn't outweigh that.
- **Synchronous.** `tickets`, `pickup`, Ship's `submit`, and `abandon` answer only when the Forgejo and git work is done. Failure partway through stays in the map's fog item "Forgejo being unreachable".
- **Versioning:** a `/v1` prefix. The server serves only its own version; any other gets `404 unsupported_version` ("this server speaks v1; restart it"), since a long-running server can outlive a CLI upgrade.

**Conventions**
- Params are exact: every param is required (nullable ones are sent as `null`), and an unknown, missing, or wrongly typed param is `400 bad_request`, naming it. This matches `taiichi.toml`, where unknown keys are an error.
- Keys are `snake_case`; timestamps are RFC 3339 in UTC; enum values are lowercase strings.
- A result always has the same keys; a field that doesn't apply is `null`.
- Every result is an object; lists are wrapped (`{"jobs": [...]}`), so a key can be added later without changing the shape.
- Ticket ids are strings (`"42"`), so Trackers with ids like `"ENG-42"` fit later.
- The server returns structured data; the CLI lays it out for people. `--json` becomes a pass-through.

**Naming the Repo and the Job**
- Every Job method takes exactly `{git_dir, job}`. `git_dir` is the clone's absolute git common dir; `job` is the per-Repo number. The server never exposes a Repo id.
- `locate` turns a worktree into a Job number. The CLI calls it only when the Job argument is left out, passing `git rev-parse --show-toplevel`; the server matches it exactly against Job worktrees. A param that branches between "job if given, else worktree" was rejected, so every method has one params shape. Reading the Job number from the branch name was rejected too: it ties the CLI to the Forgejo Tracker's naming rule and matches any checkout of the branch.

**Methods**

| Method | Params | Result |
|---|---|---|
| `status` | `git_dir` (nullable, for outside a git repo) | `{server: {version, api}, repo}`; `repo` is `null` if not initialized |
| `init` | `git_dir`, `config`, `create_labels` | `{repo, account, wrote_config}` |
| `tickets` | `git_dir` | `{tickets: [{id, title, url, labels}]}` |
| `pickup` | `git_dir`, `ticket` | `{job}` |
| `jobs` | `git_dir`, `all` | `{jobs: [Job]}`, by number |
| `locate` | `git_dir`, `worktree` | `{job}` (the number) |
| `show` | `git_dir`, `job` | `{job}`, the Job object plus `history` |
| `input` | `git_dir`, `job` | `{stage, attempt, parts: [document]}` for the current Attempt |
| `document` | `git_dir`, `job`, `stage`, `attempt` | `{document}` |
| `submit` | `git_dir`, `job`, `stage`, `attempt`, `body`, `verdict`, `title` | `{saved, job}` |
| `abandon` | `git_dir`, `job` | `{job}` |

The CLI's `path` reads `worktree` from `show`; `cat` calls `document`; `submit`'s check before opening the editor uses `show`. A Repo object (in `status` and `init`) is `{path, forge: {type, url, repo}, tracker: {type, url, repo}}`.

**Shapes**
- **Job** (returned by `pickup`, `jobs`, `show`, `submit`, `abandon`):
  ```json
  {"number": 7,
   "ticket": {"id": "42", "title": "Add a dark mode toggle", "url": "https://forgejo.example/ngs/widgets/issues/42"},
   "position": "ship", "attempt": 1, "awaiting": "shipment",
   "branch": "taiichi/42-j7-add-a-dark-mode-toggle",
   "worktree": "/home/ngs/.local/share/taiichi/repos/3/worktrees/7",
   "shipment": {"url": "https://forgejo.example/ngs/widgets/pulls/51", "last_report": "pending", "reported_at": "2026-10-17T12:03:00Z"},
   "picked_up_at": "2026-10-17T09:12:00Z", "ended_at": null}
  ```
  `position` is `plan`, `build`, `evaluate`, `ship`, `done`, or `abandoned`; `attempt` is `null` at Done and Abandoned. `awaiting` is `stage_output` (a human must submit), `shipment` (waiting on the Forge, including a Closed Shipment), or `null` once ended. `worktree` is `null` once removed. `shipment.last_report` is `pending`, `refused`, `accepted`, or `closed`: the latest Report on the Shipment, made at `reported_at`. There's no "what's next" field: the Pipeline is fixed, so the CLI derives it from `position` and `awaiting`.
- **History** (on `show` only): one entry per Attempt, oldest first, the current one last with `output: null`. Bodies come from `document`.
  ```json
  {"stage": "ship", "attempt": 1, "started_at": "...",
   "output": {"kind": "manifest", "submitted_at": "...", "meta": {"shipment_url": "...", "commit": "c07b311"}},
   "last_report": "refused", "feedback_items": 2}
  ```
  `last_report` and `feedback_items` are `null` except on Ship Attempts.
- **Document:** `{kind, stage, attempt, meta, body}`. The server's front matter comes back as `meta`, the Markdown as `body`; YAML stays a storage detail. `kind` is `ticket_snapshot`, `blueprint`, `build_notes`, `evaluation`, `manifest`, or `feedback`. A Ticket Snapshot has `null` Stage and Attempt; Feedback carries the Ship Attempt that brought it back. `cat` prints `meta` as `key: value` lines above the body.
- A Stage Input is returned as an ordered list of its parts, each a document, which the CLI prints in order under headings and a future Stage executor can take without parsing.

**Method rules**
- **`submit`** names the Stage and Attempt it was written for, filled in by the CLI when the command starts. The server refuses a mismatch with `409 job_moved` (details: the current `position` and `attempt`), so Build Notes written for Build 2 can't land on a later Attempt. `verdict` and `title` are `null` where they don't apply.
- **`abandon`** takes no such guard: a Job can be abandoned from any Position, and the CLI's confirmation prompt is seconds old.
- **`document`** with `attempt: null` returns the latest Attempt of that Stage that has a Stage Output, so `cat build` at Build 3 prints Build 2's notes.
- **`init`**: `config` is a partial `taiichi.toml` as JSON, any key the file allows, merged over the defaults when the server writes a new file. `forge.url` and `forge.repo` it leaves out still come from `origin`; the server reads `origin` and finds the clone's main working tree itself, so the CLI sends only what was typed (`{}` with no flags). If the file already exists and `config` isn't empty, `409 config_exists` instead of ignoring it. Missing labels stop the call with `409 labels_missing` (details: `labels`); the CLI asks, then calls again with `create_labels: true`. `--yes` sends `true` the first time. Rerunning `init` is already safe, so the retry needs nothing special.

**Errors**
- Body: `{"error": {"code": "...", "message": "...", "details": {...}}}`. The server writes `message`, and the CLI prints it as is. Clients rely only on `code`, never `message`. `details` is set only for codes that carry data.
- Status codes: 404 for a wrong URL *or* something named in the request that the server doesn't have. A strict reading of RFC 9110, where 404 is only about the URL, was weighed and dropped: clients branch on `code`, so the status serves people reading `curl -v` and logs, and the common reading (MDN, Twirp) is what they expect. By the same reasoning, a Forgejo failure is 502, though Taiichi isn't strictly a gateway.

| Status | Codes |
|---|---|
| 400 | `bad_request` |
| 404 | `unsupported_version`, `unknown_method`, `repo_not_initialized`, `no_such_job`, `no_such_ticket`, `no_such_document`, `not_a_job_worktree` |
| 405 (with `Allow: POST`), 413 (over Plug's 8 MB default), 415 | `method_not_allowed`, `too_large`, `unsupported_media_type` |
| 409 (state the user can fix) | `job_moved`, `job_ended`, `not_awaiting_output`, `ticket_not_eligible`, `ticket_has_active_job`, `branch_taken`, `worktree_dirty`, `head_moved`, `forge_location_changed`, `forge_location_taken`, `labels_missing`, `config_invalid`, `token_missing`, `config_exists` |
| 422 (invalid content) | `invalid_output` (empty body, a Verdict missing or not allowed, an empty title), `forge_unknown` (no HTTP(S) `origin` and no Forge in `config`) |
| 500 | `internal` (anything else, including a git failure) |
| 502 | `forgejo_failed` |

This is v1's starting list; codes are added as the spec needs them. Each one matches a check settled in an earlier ticket.

No ADR: the choices are reversible and documented here.

## Comments

**From [Server internals](12-server-internals.md):** Job numbers are server-wide, so Job methods (`show`, `input`, `document`, `submit`, `abandon`) take `{job, …}` without `git_dir`, and `locate` takes `{worktree}`. `jobs`, `tickets`, `pickup`, `init`, and `status` keep `git_dir`. Worktree paths are `…/taiichi/worktrees/<job>` (the example's `repos/3/worktrees/7` is out of date). Reads come straight from the database, and a Ship Attempt's `shipment.last_report` and `reported_at` are stored by the poller.
