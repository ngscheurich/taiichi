# Forgejo API for the Tracker and Forge adapters

Research for Taiichi v1, where one self-hosted Forgejo instance is both the
Tracker (its issues) and the Forge (its pull requests). It answers what
Forgejo's API offers for each adapter, how polling and auth work, and what
constrains the design.

## Version and sources

- **Primary version: Forgejo v17.0.0** (tag `v17.0.0`, commit `0d791421`,
  released 15 October 2026), the latest stable release. Supported series on
  17 October 2026 are 17.0 (until 28 Jan 2027), 16.0 (until 29 Oct 2026) and
  **15.0 LTS (until 15 Jul 2027)**
  ([releases](https://forgejo.org/releases/),
  [release schedule](https://forgejo.org/docs/v17.0/admin/release-schedule/)).
- The most design-relevant claims were re-checked against **v15.0.10 (LTS)**
  source and hold there too: the self-review ban, label-filter semantics, the
  scope split between `/issues` and `/pulls`, the "poster can't be reviewer"
  rule, and the closing-keyword permission rule. Repo-specific tokens exist
  since v15.0.0 (see [Auth](#auth)). Everything else was read only on v17.0.0.
- The API is stable within a major version; breaking changes only come with a
  new major number
  ([API usage](https://forgejo.org/docs/v17.0/user/api-usage/)). Forgejo
  ships a major every quarter, so the adapter should record which major it was
  tested against.
- Sources: the OpenAPI 2.0 spec
  [`templates/swagger/v1_json.tmpl`][swagger] (served live at
  `/swagger.v1.json` and `/api/swagger` on any instance), the Go handlers
  behind it, and the v17.0 docs on forgejo.org. Where the spec and the code
  disagree, the code is quoted and the gap is called out.

Source links below are relative to
`https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/`. All API paths are
under `/api/v1`.

[swagger]: https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/templates/swagger/v1_json.tmpl

## Summary for design

1. **The pull request's author can't approve it or request changes on it.**
   The API returns 422 (`approve your own pull is not allowed` /
   `reject your own pull is not allowed`), and the poster also can't be
   *requested* as a reviewer. Taiichi must open Shipments as a different
   account (a bot user) from the human who reviews them.
2. **"Refused" isn't a pull request state.** A Refused Shipment shows up as
   a review with `state: "REQUEST_CHANGES"` from `GET /pulls/{index}/reviews`.
   The pull request itself stays `open`. A reviewer can submit several
   reviews, so the latest non-dismissed review per reviewer decides. A
   request-changes review must have a body.
3. **"Accepted" is `merged: true`** on the pull request (or
   `pull_request.merged` on the issue view of it). "Closed without merging" is
   `state: "closed"` with `merged: false`, a third outcome Taiichi must
   handle.
4. **`Closes #12` in the pull request's title or body closes issue 12 when the
   pull request is merged**, with the merger as the actor. The keyword list is
   configurable per instance; matching is case-insensitive and allows a colon
   (`Closes: #12`), but each issue reference needs its own keyword
   (`Closes #12, #13` closes only 12).
5. **The `labels` filter on `GET /repos/{owner}/{repo}/issues` means "all of",
   not "any of" as the spec says.** An unknown label name is silently
   dropped, so a filter on one misspelled label returns *every* open issue.
   The adapter should check that its labels exist.
6. **Cheapest poll:** `GET /repos/{owner}/{repo}/issues?type=pulls&state=all&since=<t>`.
   It's served from the database, filters by `updated_at`, and each item
   carries `pull_request.merged`. Any comment, review, label change, push or
   state change bumps `updated_at`. `GET /pulls` has no `since` filter and
   does several git operations per pull request, so it's heavier.
7. **No API rate limiting** exists in Forgejo itself. The only limit is page
   size: at most 50 items a page by default (`[api] MAX_RESPONSE_ITEMS`).
8. **Scopes split by route, not by concept:** everything under
   `/repos/{o}/{r}/issues/*` (including labels and comments *on pull
   requests*) needs `read:issue`/`write:issue`; everything under `/pulls/*`
   and git-over-HTTPS needs `read:repository`/`write:repository`. A token
   restricted to specific repositories (v15+) can hold exactly these four
   scopes, which is all v1 needs.
9. **Assignees can only be replaced as a whole set** (`PATCH` the issue with
   `assignees`); there's no add or remove endpoint, so assigning is
   read-modify-write.
10. **Exclusive ("scoped") labels** such as `status/building` remove other
    labels in the same scope when added, which suits pipeline-stage labels.

## Tracker adapter

### Fetching an issue

`GET /repos/{owner}/{repo}/issues/{index}` returns an `Issue` with `title`,
`body`, `labels[]`, `assignees[]`, `state` (`open`/`closed`), `user`,
`updated_at`, `closed_at`, `html_url`, and `pull_request` (non-null when the
"issue" is actually a pull request) ([swagger L11426][swagger]; `Issue`
definition in the same file). Pull requests share the issue number space, so
the Tracker adapter must ignore or reject items whose `pull_request` is set.

Comments: `GET /repos/{owner}/{repo}/issues/{index}/comments` with optional
`since`/`before` ([swagger L12016][swagger]).

### Listing issues by label

`GET /repos/{owner}/{repo}/issues` takes `state` (`open` default, `closed`,
`all`), `labels` (comma-separated names), `type` (`issues`/`pulls`), `q`,
`milestones`, `since`/`before` (RFC 3339, on `updated_at`), `created_by`,
`assigned_by`, `mentioned_by`, `sort`, `page`, `limit`
([swagger L10509][swagger]). The response carries `X-Total-Count` and a
`Link` header ([API usage](https://forgejo.org/docs/v17.0/user/api-usage/);
`ctx.SetLinkHeader` in
[`routers/api/v1/repo/issue.go`](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue.go#L342)).

Label filter behaviour, from the code rather than the spec:

- Names are resolved to IDs among the repository's labels (and the org's, if
  the owner is an org). Names that don't exist are dropped
  ([`issue.go` L447–462](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue.go#L447),
  [`models/issues/label.go` L372](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/label.go#L372)).
  If none resolve, no label filter is applied at all.
- With no search keyword, the query always goes to the database indexer
  ([`modules/indexer/issues/indexer.go` L333–344](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/indexer/issues/indexer.go#L333)),
  which requires the issue to carry **every** listed label: it compares
  `count(*)` of matching `issue_label` rows to the number of labels
  ([`models/issues/issue_search.go` L127–149](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/issue_search.go#L127)).
  The bleve and meilisearch indexers also AND them. The spec's "any of this
  labels" is wrong.
- `type=issues` excludes pull requests.

Org-wide or cross-repo listing: `GET /repos/issues/search` takes similar
filters ([swagger L5139][swagger]); not examined in detail.

### Adding and removing labels

| Operation | Endpoint | Notes |
| --- | --- | --- |
| Add | `POST /repos/{o}/{r}/issues/{index}/labels` body `{"labels": [...]}` | Labels as all names or all IDs, not mixed (400). Unknown names are silently ignored. Returns the issue's full label list. |
| Replace all | `PUT /repos/{o}/{r}/issues/{index}/labels` | Same body. |
| Remove one | `DELETE /repos/{o}/{r}/issues/{index}/labels/{identifier}` | `identifier` is a name, falling back to an ID. 422 if no such label; 204 on success. |
| Clear | `DELETE /repos/{o}/{r}/issues/{index}/labels` | |

Sources: [swagger L12479, L12685][swagger];
[`routers/api/v1/repo/issue_label.go` L123–211, L332–390](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue_label.go#L123).
All need write access to issues (or pull requests, for a pull request) on the
repository (`CanWriteIssuesOrPulls`, same file).

Exclusive labels: a label marked `exclusive` whose name contains `/` has a
*scope* (the part before the last `/`). Adding it removes any other label in
the same scope from the issue
([`models/issues/label.go` L195](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/label.go#L195),
[`models/issues/issue_label.go` L66](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/issue_label.go#L66);
see also [labels docs](https://forgejo.org/docs/v17.0/user/labels/)).
A `status/planning` → `status/building` move is then one `POST`. Not
confirmed: whether swapping is atomic as seen by a concurrent reader.

### Assigning users

There's no `/issues/{index}/assignees` endpoint (absent from the spec). The
only way is `PATCH /repos/{o}/{r}/issues/{index}` with
`{"assignees": ["login", ...]}`, which **replaces** the set; `[]` clears it
([`issue.go` L895–913](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue.go#L895)).
Constraints:

- The caller needs write access to issues, otherwise the request is rejected
  ([`routers/api/v1/permissions/req_edit_issue.go` L28–31](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/permissions/req_edit_issue.go#L28)).
- Each assignee must have write access to some unit of the repository or read
  access to its pull requests; orgs can't be assigned
  ([`models/perm/access/repo_permission.go` L411](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/perm/access/repo_permission.go#L411),
  checked in
  [`services/issue/issue.go` L206](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/issue/issue.go#L206)).
  A bad login surfaces as a 500 from this endpoint, not a 422.

### Commenting

`POST /repos/{o}/{r}/issues/{index}/comments` with `{"body": "..."}` returns
201 and the `Comment` ([swagger L12016][swagger];
[`issue_comment.go` L358](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue_comment.go#L358)).
The same endpoint comments on a pull request's conversation.

### Closing issues

Directly: `PATCH /repos/{o}/{r}/issues/{index}` with `{"state": "closed"}`.
Returns 412 if the issue has open dependencies (blocking issues)
([`issue.go` L925–946](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/issue.go#L925)).

Via closing keywords in a pull request:

- When a pull request is created or its title/body edited, Forgejo parses
  references in both and records a cross-reference with an action (close or
  reopen) on each referenced issue
  ([`models/issues/issue_xref.go` L62–75, L140–176](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/issue_xref.go#L62)).
  The issue shows a notice that it will be closed
  ([linked references](https://forgejo.org/docs/v17.0/user/linked-references/)).
- On merge, `handleCloseCrossReferences` closes each issue with a "close"
  action, acting as the merger
  ([`services/pull/merge.go` L288–313](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/pull/merge.go#L288)).
  Only the latest reference per issue counts
  ([`issue_xref.go` `ResolveCrossReferences`](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/issue_xref.go#L337)).
- Keywords default to `close, closes, closed, fix, fixes, fixed, resolve,
  resolves, resolved` (reopen: `reopen, reopens, reopened`), set by
  `[repository.pull-request] CLOSE_KEYWORDS`
  ([`modules/setting/repository.go` L234](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/setting/repository.go#L234),
  [config cheat sheet](https://forgejo.org/docs/v17.0/admin/config-cheat-sheet/)).
- Pattern: case-insensitive keyword, optional `:`, exactly one space, then the
  reference: `(?i)(?:\s|^|\(|\[)(keywords):? $` matched against the text
  before the reference
  ([`modules/references/references.go` L136–143, L572–587](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/references/references.go#L136)).
  So `Closes #12, #13` closes only #12; write `Closes #12, closes #13`.
- Who may set a close action: the reference is kept if the author can write
  issues, is the issue's poster, **or** it's in a new pull request's
  description in the same repository. References in later *comments* by
  someone without write access lose their action
  ([`issue_xref.go` L216–234](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/issue_xref.go#L216)).
  The docs say the merger must then have permission to close the issue; the
  merge path doesn't re-check that (see Unconfirmed).
- Only pull requests (and commits) can close; only issues can be closed. A
  reference to another pull request never acts (same file, L211–214).
- Commit messages with keywords also close issues when pushed to the default
  branch, if the pusher may close them
  ([`services/issue/commit.go` L150–195](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/issue/commit.go#L150)).
  So a merged commit message containing `Fixes #12` closes #12 even if the
  pull request body doesn't mention it.
- Instances can add a merge-message template with `${ClosingIssues}`
  ([`merge.go` L144–160](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/pull/merge.go#L144),
  [merge message templates](https://forgejo.org/docs/v17.0/user/merge-message-templates/)).

Design consequence: Taiichi can rely on `Closes #N` in the Shipment body and
needn't close the Ticket itself. To be safe against config changes it can
also verify, after observing the merge, that the issue is closed.

## Forge adapter

### Pushing a branch

Plain git; Forgejo adds no API step.

- **HTTPS:** basic auth with a username and an access token as the password.
  The token needs `read:repository` to fetch and `write:repository` to push
  ([`routers/web/repo/githttp.go` L156](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/web/repo/githttp.go#L156)
  maps the git access mode onto the repository scope).
  Repo-specific tokens are enforced on git operations since v15.0.0
  ([15.0.0 release notes, PR #11452](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/release-notes-published/15.0.0.md)).
- **SSH:** a user SSH key (acts as that user) or a deploy key added to the
  repository (read-only or read-write). Token scopes don't apply to SSH.
  Deploy keys can't call the API, so Taiichi would still need a token.
  (Deploy-key behaviour from general Forgejo knowledge; not re-read in source.)
- **AGit alternative:** `git push origin HEAD:refs/for/main/<topic> -o
  title="..." -o description="{base64}..."` pushes and opens a pull request
  in one step, with no branch in the repository and no write permission
  needed on it
  ([AGit docs](https://forgejo.org/docs/v17.0/user/agit-support/)). The
  pull request's head ref is then `refs/pull/N/head`. Worth considering if the
  bot account should not be able to write branches.

### Opening a pull request

`POST /repos/{o}/{r}/pulls` with `head`, `base`, `title` (all required),
`body`, `assignees`, `labels` (**IDs only**, unlike issues), `milestone`,
`due_date` ([swagger L14870][swagger];
[`modules/structs/pull.go` L80](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/structs/pull.go#L80)).

- `base` is required; there's no "default branch" shortcut. Read it from
  `GET /repos/{o}/{r}` → `default_branch`.
- `head` is `branch` or `owner:branch` (fork) (`parseCompareInfo`,
  [`pull.go` L1085](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/pull.go#L1085)).
- If an unmerged pull request already exists for the same head and base, the
  API returns **409** (`CreatePullRequest`,
  [`pull.go` L399](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/pull.go#L399)).
  `GET /repos/{o}/{r}/pulls/{base}/{head}` finds it, which makes "open
  Shipment" idempotent.
- `head == base` → 422.

### Reading merge state

`GET /repos/{o}/{r}/pulls/{index}` returns `state`, `merged`, `merged_at`,
`merged_by`, `merge_commit_sha`, `mergeable`, `draft`, `head.sha`,
`requested_reviewers`, `review_comments` count ([swagger L15132][swagger];
`PullRequest` definition). `GET /pulls/{index}/merge` answers 204 (merged) or
404 (not merged) ([swagger L15443][swagger]).

| Taiichi outcome | Forgejo signal |
| --- | --- |
| Accepted | `merged: true` |
| Refused | latest non-dismissed review from a reviewer has `state: "REQUEST_CHANGES"`; pull request stays `open` |
| (unnamed) | `state: "closed"`, `merged: false`: closed without merging |

### Reading reviews

`GET /repos/{o}/{r}/pulls/{index}/reviews` lists reviews oldest-first
([swagger L15710][swagger];
[`models/issues/review_list.go` L123–133](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/review_list.go#L123)).
Each `PullReview` has `id`, `user`, `state`, `body`, `commit_id`, `stale`,
`dismissed`, `official`, `comments_count`, `submitted_at`, `updated_at`.

- `state` values: `APPROVED`, `REQUEST_CHANGES`, `COMMENT`, `PENDING`,
  `REQUEST_REVIEW` (a pending review request, not a review), and `""`
  ([`modules/structs/pull_review.go` L13–26](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/structs/pull_review.go#L13),
  mapped in
  [`services/convert/pull_review.go` L48–59](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/convert/pull_review.go#L48)).
- Other users' `PENDING` (draft) reviews are hidden unless the caller is a
  site admin (same file, L65–79).
- The overall "request changes" message is the review's `body`. A
  request-changes review must have a non-empty body; approve and comment
  needn't
  ([`routers/api/v1/repo/pull_review.go` L644–690](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/pull_review.go#L644)).
- Inline comments: `GET /pulls/{index}/reviews/{id}/comments` returns
  `PullReviewComment`s with `path`, `position`/`original_position`,
  `diff_hunk`, `body`, `commit_id`, `resolver` ([swagger L15985][swagger]).
  There's no "all review comments on a pull request" endpoint; it's one call
  per review. General conversation comments come from
  `/issues/{index}/comments`. `/issues/{index}/timeline` mixes everything
  ([swagger L13544][swagger]).
- A push to the pull request marks all its reviews `stale: true`; if the
  branch protection rule has "dismiss stale approvals", approvals (not
  request-changes reviews) are also dismissed
  ([`services/pull/pull.go` L396–406](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/pull/pull.go#L396),
  [`models/issues/review.go` L561–583](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/review.go#L561)).
  A request-changes review therefore stays in force after Taiichi pushes a
  fix, until the reviewer reviews again or it's dismissed.
- Omitting `page` returns all reviews unpaginated
  ([`review_list.go` L126](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/review_list.go#L126)).

### Who may review or approve

- **The pull request's poster can't approve or request changes** on it via
  the API (422) or the web UI
  ([`routers/api/v1/repo/pull_review.go` L655–668](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/pull_review.go#L655),
  [`routers/web/repo/pull_review.go` L263](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/web/repo/pull_review.go#L263)).
  The poster may still leave `COMMENT` reviews. Same in v15.0.10.
- **The poster can't be requested as a reviewer** ("poster of pr can't be
  reviewer"); a requested reviewer needs read access to pull requests
  ([`services/issue/assignee.go` L112–126](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/issue/assignee.go#L112)).
- Anyone who can read the repository's code and pull requests may submit a
  review (route guards in
  [`routers/api/v1/api.go` L1020–1061](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/api.go#L1020)).
  Whether it's **official** depends on branch protection: with no rule, users
  with write access to code are official; with a rule and an approvals
  allowlist, only listed users/teams are
  ([`models/issues/review.go` L259–288](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/review.go#L259),
  [`models/git/protected_branch.go` L177](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/git/protected_branch.go#L177)).
- Branch protection rules carry `RequiredApprovals`,
  `EnableApprovalsWhitelist`, `BlockOnRejectedReviews`,
  `BlockOnOfficialReviewRequests`, `DismissStaleApprovals`,
  `IgnoreStaleApprovals`, `EnableMergeWhitelist`
  ([`protected_branch.go` L43–57](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/git/protected_branch.go#L43);
  [protection docs](https://forgejo.org/docs/v17.0/user/protection/)).
  Taiichi can read but needn't depend on these: Accepted/Refused are decided
  by merge and review, which a human does.

Design consequence: Taiichi needs **its own Forgejo user** (a bot) to open
Shipments, distinct from every human reviewer. If it used a human's token, that
human could never Refuse or approve the Shipment.

## Polling

Candidate endpoints, cheapest first:

1. **`GET /repos/{o}/{r}/issues?type=pulls&state=all&since=<last poll>&sort=recentupdate&limit=50`**
   - Served from the database, never from git
     ([`indexer.go` L336–344](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/indexer/issues/indexer.go#L336)).
   - `since` filters on the issue's `updated_unix`. `CreateComment` bumps it
     for every timeline comment
     ([`models/issues/comment.go` L1103–1137](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/comment.go#L1103)),
     and submitting a review creates a review comment
     ([`models/issues/review.go` L479](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/issues/review.go#L479)),
     so reviews, comments, label and assignee changes, pushes, closes and merges
     all surface here.
   - Each item's `pull_request` has `merged` and `merged_at`
     ([`services/convert/issue.go` L103–111](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/convert/issue.go#L103)),
     so "Accepted?" needs no further call. "Refused?" needs one
     `GET /pulls/{index}/reviews` for each changed pull request.
   - Needs the `issue` scope, even though it lists pull requests (it's under
     the issue route group,
     [`api.go` L1142–1256](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/api.go#L1142)).
   - The same call with `type=issues&labels=<ready label>` polls the Tracker.
2. **`GET /repos/{o}/{r}/pulls?state=open`** takes `state`, `sort`,
   `milestone`, `labels` (IDs), `poster`, `base`, `head`, but **no `since`**
   ([swagger L14870][swagger];
   [`pull.go` L49](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/pull.go#L49)).
   Converting each pull request opens the base and head git repositories and
   resolves branch commits
   ([`services/convert/pull.go` L24–250](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/convert/pull.go#L24)),
   so it costs several git operations per item. Useful to filter by `poster`
   (the bot) when the bot's pull requests are a small share of the
   repository's.
3. **Webhooks** as a push alternative: events include `pull_request`,
   `pull_request_review_approved`, `pull_request_review_rejected`,
   `pull_request_review_comment`, `pull_request_sync`, `issues`,
   `issue_label`, `issue_comment`
   ([`modules/webhook/type.go` L12–30](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/webhook/type.go#L12),
   [webhooks docs](https://forgejo.org/docs/v17.0/user/webhooks/)).
   They need Taiichi to be reachable from Forgejo; polling still serves as a
   catch-up.

Limits and caching:

- **No rate limiting.** Forgejo's source has no API rate limiter; the only
  rate-limit code is in the GitHub migration client
  (`grep` for `ratelimit`/`throttl` across `modules`, `routers`, `services`).
  The config cheat sheet lists none. A reverse proxy in front of the user's
  instance could add one; that couldn't be checked.
- Page size: default 30, maximum 50 (`[api] DEFAULT_PAGING_NUM`,
  `MAX_RESPONSE_ITEMS`); larger `limit` values are clamped
  ([`modules/setting/api.go` L24–25](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/modules/setting/api.go#L24),
  [`services/convert/utils.go` L15](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/services/convert/utils.go#L15)).
  `GET /settings/api` reports the instance's values
  ([API usage](https://forgejo.org/docs/v17.0/user/api-usage/)).
- **No ETag or conditional requests** on issue or pull request JSON; the only
  ETag handling in `routers/api` is for raw file and LFS downloads
  ([`routers/api/v1/repo/file.go` L138](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/repo/file.go#L138)).
  `since` is the only way to make a poll cheap.

## Auth

### Credential types

| Type | How it's sent | Scopes | Notes |
| --- | --- | --- | --- |
| Personal access token | `Authorization: token …` or `Bearer …`, or as the basic-auth password | Yes (required) | Repository access: all, public-only, or **specific repositories** (v15+). Created in the UI or via `POST /users/{name}/tokens` with basic auth. |
| OAuth2 access token | `Authorization: Bearer …` | **No**: full access as the user | Forgejo as OAuth2 provider. |
| Authorized Integration | JWT from an external OIDC issuer | Yes, plus repo restriction | New in v16.0; short-lived, no static secret. |
| Forgejo Actions token | `${{ forgejo.token }}` | Fixed, limited | Only inside Actions runs. |
| Basic auth with password | `-u user:pass` (+ `X-Forgejo-OTP` with 2FA) | n/a | Needed to mint tokens via API. |

Sources: [API usage](https://forgejo.org/docs/v17.0/user/api-usage/),
[token scopes](https://forgejo.org/docs/v17.0/user/token-scope/),
[OAuth2 provider](https://forgejo.org/docs/v17.0/user/oauth2-provider/)
("OAuth2 scopes are not yet implemented"),
[Authorized Integrations](https://forgejo.org/docs/v17.0/user/authorized-integrations/),
[`models/auth/access_token_resource.go`](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/models/auth/access_token_resource.go),
[16.0.0 release notes](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/release-notes-published/16.0.0.md).

A personal access token for the bot user, restricted to the specific
repositories Taiichi works on, is the fit for v1. Its permissions are the
**intersection** of the token's scopes and the bot user's repository
permissions, so the bot must also be a collaborator with write access.

### Scopes per operation

Scope level comes from the HTTP method: `GET` needs `read:*`; `POST`, `PUT`,
`PATCH` and `DELETE` need `write:*`
([`api.go` L293–300](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/api.go#L293)).
The category comes from the route group: `/repos/{o}/{r}/issues/*`,
`/labels`, `/milestones` → **issue**; other `/repos/{o}/{r}/*` including
`/pulls/*` → **repository**
([`api.go` L1020–1131 and L1142–1256](https://codeberg.org/forgejo/forgejo/src/tag/v17.0.0/routers/api/v1/api.go#L1142);
[token scopes](https://forgejo.org/docs/v17.0/user/token-scope/)).

| Operation | Endpoint | Scope | Repo permission |
| --- | --- | --- | --- |
| Get issue / list issues / comments | `GET …/issues…` | `read:issue` | read issues |
| Poll pull requests via issues list | `GET …/issues?type=pulls` | `read:issue` | read pull requests |
| Add/remove labels, assign, comment, close | `POST/DELETE/PATCH …/issues/{index}…` | `write:issue` | write issues (pull requests, for a PR) |
| Read repo (default branch) | `GET /repos/{o}/{r}` | `read:repository` | read |
| Fetch / push over HTTPS | git smart HTTP | `read:repository` / `write:repository` | read / write code |
| Open pull request | `POST …/pulls` | `write:repository` | read code + pull requests |
| Read pull request, reviews, review comments | `GET …/pulls/…` | `read:repository` | read code |
| Labels/comments on a pull request | `…/issues/{index}/…` | `issue` scope | write pull requests |

A specific-repository token may only hold `read/write:repository` and
`read/write:issue`, exactly this set, and can't perform repository-admin
actions ([token scopes](https://forgejo.org/docs/v17.0/user/token-scope/)).

## Unconfirmed or not checked

- **Live behaviour on the user's instance:** no request was made against it.
  Its version, `CLOSE_KEYWORDS`, `MAX_RESPONSE_ITEMS`, branch protection and
  any reverse-proxy rate limit are unknown. `GET /api/v1/version` and
  `GET /api/v1/settings/api` will tell.
- **Closing keywords and the merger's permission:** the docs say a
  description-only reference closes the issue "only if the merger … has
  permissions"; `handleCloseCrossReferences` in v17.0.0 calls `ChangeStatus`
  without a permission check. In practice a merger has write access, so this
  probably doesn't matter.
- **Deploy keys** (read-write SSH push) weren't re-read in source.
- **`/repos/issues/search`** (cross-repo) wasn't examined beyond its
  existence.
- **Whether exclusive-label swaps are atomic** as seen by a concurrent
  poller wasn't checked.
- **v16.x** wasn't checked separately; the claims verified on both v15.0.10
  and v17.0.0 presumably hold there.
- **Forgejo v17.0.0 release notes** aren't in the tag's
  `release-notes-published/` (it stops at 16.0.5), so changes new in 17.0
  weren't reviewed systematically.
