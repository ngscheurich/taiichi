# Forgejo API for the Tracker and Forge

Type: research
Status: resolved

## Question

What does Forgejo's API offer for the two adapters v1 needs, and how?

- **Tracker:** fetching an issue (title, body, labels, assignees, state); listing issues by label; adding and removing labels; assigning users; commenting; closing issues, including closing keywords such as `Closes #12` in a pull request.
- **Forge:** pushing a branch (git over SSH or HTTPS) and opening a pull request against the default branch; reading a pull request's merge state and reviews, including "request changes" and the review comments; anything that limits who may review or approve a pull request, such as the author reviewing their own.
- **Polling:** which endpoints and filters make polling open pull requests cheap, and any rate limits.
- **Auth:** access token types and the scopes each operation needs.

Primary sources: Forgejo's documentation and its OpenAPI spec. Record the Forgejo version the findings apply to.

## Answer

Checked against Forgejo v17.0.0 source and docs; the key claims also hold in the v15 long-term-support release. The user's instance version is unconfirmed (`GET /api/v1/version` will tell).

- **Self-review is blocked.** A pull request's author can't approve it or request changes on it (422), and can't be requested as a reviewer. Whoever publishes Shipments can't also record a verdict on them.
- **Refused is a review, not a pull request state:** `state: "REQUEST_CHANGES"` from `GET /pulls/{index}/reviews`; the latest non-dismissed review per reviewer decides. Request-changes reviews require a body; inline comments need one extra call per review. A push marks earlier reviews stale, but a request-changes review stays in force until re-reviewed or dismissed.
- **Accepted is `merged: true`.** A pull request can also end `closed` with `merged: false`.
- **Closing keywords work:** `Closes #12` in the title or body closes the issue on merge. One keyword per issue (`Closes #12, #13` closes only #12); the keyword list is configurable per instance.
- **Label filtering:** `GET /issues?labels=a,b` matches issues with *all* the labels (the spec wrongly says any). Unknown label names are silently dropped, so a misspelled single-label filter returns every open issue; adding an unknown label is silently ignored. Exclusive (scoped) labels such as `status/x` replace others in the same scope.
- **Assignees** can only be replaced as a whole set via `PATCH` on the issue: read, modify, write.
- **Cheapest poll:** `GET /repos/{o}/{r}/issues?type=pulls&state=all&since=<t>`. It is database-only, any comment, review, label change, push or merge bumps the timestamp, and each item says whether it's merged. `GET /pulls` has no `since` and is costlier. Webhooks exist if push delivery is wanted later.
- **No built-in rate limits**, no ETags; pages default to 50 items.
- **Scopes follow the URL:** `/issues/*` (including labels and comments on pull requests, and the poll above) needs `issue`; `/pulls/*` and git over HTTPS need `repository`. Read for `GET`, write otherwise. A repository-limited personal access token with `read:issue`, `write:issue`, `read:repository`, `write:repository` covers v1.
- **Creating a pull request** needs `base` set explicitly (from `default_branch`), takes label IDs not names, and returns 409 if one already exists for the same branches.

Findings: `docs/research/forgejo-api.md` on branch `research/forgejo-api` (commit `9a0dc8b`). The branch is local only; the push failed because the sandbox has no `ssh`.
