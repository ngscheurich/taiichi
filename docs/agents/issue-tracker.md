# Issue tracker: Forgejo

Issues and specs for this repo live as issues in the repo `ngscheurich/taiichi` on a Forgejo instance running on the user's host. There is no Forgejo CLI in the sandbox, so every operation goes through the API with `scripts/forgejo`, a thin `curl` wrapper, and `jq` to read the replies.

## Access

- **From the sandbox:** `http://host.microsandbox.internal:3000` (the wrapper's default).
- **In the browser:** the user opens the same instance as `http://localhost:3000`, which is also the host in the URLs Forgejo returns (`html_url`). Give the user those links as they are.
- **Token:** the wrapper reads `FORGEJO_TOKEN`, a personal access token with `read:issue` and `write:issue` scope. If it isn't set, stop and ask the user for it rather than working around it. Never print it or write it to a file.
- **Who you act as:** every write is made as the token's account. `scripts/forgejo GET /user | jq -r .login` names it.

## Calling the API

```sh
scripts/forgejo METHOD PATH [BODY]
```

`PATH` is relative to the repo (`issues/12`), or to `/api/v1` when it starts with `/` (`/user`). `BODY` is JSON; pass `-` to read it from stdin. Build bodies with `jq` so Markdown is escaped correctly, for example from a file you wrote in the scratchpad:

```sh
jq -n --arg title "Gates" --rawfile body "$SCRATCH/body.md" '{title: $title, body: $body}' \
  | scripts/forgejo POST issues -
```

Lists return at most 50 items per page. Pass `limit=50` and keep asking for `page=2`, `page=3` and so on until a page comes back short.

## Conventions

- **Create an issue:** `POST issues` with `{title, body}`. Creating takes label and milestone **ids**, not names, so add labels by name afterwards (below), or look the ids up with `GET labels?limit=50` and `GET milestones?state=all`.
- **Read an issue:** `scripts/forgejo GET issues/<n> | jq '{number, title, state, body, labels: [.labels[].name], assignees: [.assignees[]?.login], milestone: .milestone.title}'`, and its comments with `GET issues/<n>/comments | jq '[.[] | {user: .user.login, body}]'`.
- **List issues:** `GET 'issues?state=open&type=issues&limit=50'`, adding `labels=<name>` and `milestones=<name>` filters. Filter on one label at a time: Forgejo silently drops label names it doesn't know, so a misspelt label returns every issue instead of none. `type=issues` matters because pull requests share the list and the number space.
- **Comment on an issue:** `POST issues/<n>/comments` with `{body}`.
- **Apply / remove labels:** `POST issues/<n>/labels` with `{"labels": ["<name>"]}` / `DELETE issues/<n>/labels/<name>`. Don't use `PUT` on the labels list: it replaces every label, including ones you weren't told about.
- **Close / reopen:** `PATCH issues/<n>` with `{"state": "closed"}` or `{"state": "open"}`. Post any closing explanation as a comment first.
- **Edit a body:** fetch it to a file (`GET issues/<n> | jq -r .body > "$SCRATCH/<n>.md"`), edit the file, then `PATCH issues/<n>` with `{body}`. Other sessions may edit the same issue, so fetch again just before you write and redo the edit if the body changed.
- **Cross-references:** write `#<n>`, for example `the v1 spec (#24)`. Forgejo links it and lists the reference on the other issue. Link to repo files by path from the repo root, for example `[GLOSSARY.md](/ngscheurich/taiichi/src/branch/main/GLOSSARY.md)`.
- **Efforts:** each feature or effort is a milestone named by its slug (for example `taiichi-v1`). Its map, spec and tickets carry that milestone.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests; `/triage` reads this flag.)_

When set to `yes`, PRs go through the same labels and states as issues. A pull request is an issue with code attached, so the issue endpoints for comments, labels and closing work on it too. Read one with `GET pulls/<n>`, and its diff with `GET pulls/<n>.diff`. Issues and pull requests share one number space, so a bare `#42` may be either; `GET issues/<n>` answers for both, and `.pull_request` is non-null for a pull request.

## When a skill says "publish to the issue tracker"

Create a Forgejo issue in the effort's milestone, creating the milestone first if it doesn't exist (`POST milestones` with `{title}`).

## When a skill says "fetch the relevant ticket"

Read it as in **Read an issue** above, comments included. The user will normally pass the issue number.

## Wayfinding operations

Used by `/wayfinder`. The **map** is a single issue, and the **child** tickets are issues in the same milestone.

- **Map:** an issue labelled `wayfinder:map`, holding the Destination / Notes / Decisions so far / Not yet specified / Out of scope body, in the effort's milestone.
- **Child ticket:** an issue in the map's milestone, with `Part of #<map>` as the first line of its body and one `wayfinder:<type>` label (`research`/`prototype`/`grilling`/`task`). Forgejo has no sub-issues, so the milestone is what groups a map's tickets.
- **Blocking:** Forgejo's native issue dependencies, which show in the issue's sidebar. Add an edge with `scripts/forgejo POST issues/<child>/dependencies '{"owner": "ngscheurich", "repo": "taiichi", "index": <blocker>}'`, and read a ticket's blockers with `GET issues/<child>/dependencies`. A ticket is unblocked when every blocker is closed. Forgejo refuses to close an issue while it has open blockers.
- **Frontier query:** list the open issues in the map's milestone (`GET 'issues?state=open&type=issues&milestones=<effort>&limit=50'`) and keep those with a `wayfinder:<type>` label other than `wayfinder:map` and no assignee. Drop any whose `GET issues/<n>/dependencies` includes an open issue. The lowest number wins.
- **Claim:** `PATCH issues/<n>` with `{"assignees": ["<login>"]}`, using the token's login from **Access**. This is the session's first write.
- **Resolve:** post the answer as a comment (`POST issues/<n>/comments`), close the issue, then add a context pointer (gist + `#<n>`) to the map's Decisions so far, editing the map's body as in **Edit a body**.

Tickets migrated from the old `.tracker/` files hold their answer in the body under `## Answer`, and notes left by later tickets under `## Comments`, rather than as Forgejo comments. Read both places.
