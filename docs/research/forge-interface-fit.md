# Forge interface fit: GitHub and GitLab

Researched 2026-10-17. This note asks whether Taiichi's two-operation Forge adapter, designed against Forgejo for v1, also fits GitHub and GitLab.

The two operations under test:

1. **Publish**: open a Shipment (pull request or merge request) for a Job's branch and return a Manifest.
2. **Report**: poll the Shipment and report **Accepted** (merged) or **Refused** (changes requested), with the reviewer's feedback, which goes back to Build.

## Answer

The two-operation shape holds on all three Forges. Every Forge exposes one call to open a Shipment, a merged flag to detect Accepted, and a request-changes review state to detect Refused. GitLab has had a real "Request changes" review state since 16.11, so the old gap is gone.

The interface needs four changes to be accurate, though. None of them adds a third operation:

- **Report needs more outcomes than two.** It must also return *Pending* (no verdict yet, including "approved but not merged") and *Closed* (closed without merging).
- **Report must be scoped to the latest Publish.** A change request stays in force after new commits are pushed, so a naive poll would report the same Refusal again after Build reworks the branch. The Manifest must record the published head SHA and a feedback cursor.
- **Publish must also update an existing Shipment.** Rework after a Refusal should push to the same branch and Shipment, not open a new one.
- **Feedback is spread over two or three API resources on every Forge.** The adapter has to merge them into one shape. On GitLab it also has to rebuild "the review", because GitLab's REST API has no review object.

The rest of this note gives the evidence.

## Summary table

| Concern | GitHub | GitLab | Forgejo |
|---|---|---|---|
| Publish | `POST /repos/{o}/{r}/pulls` | `POST /projects/:id/merge_requests` | `POST /repos/{o}/{r}/pulls` |
| Draft at creation | `draft: true` | Title prefix `Draft:` | Title prefix `WIP:` / `[WIP]` (configurable) |
| Accepted signal | `merged == true` (or `GET …/merge` → 204) | `state == "merged"` | `merged == true` |
| Closed unmerged | `state == "closed"`, `merged == false` | `state == "closed"` | `state == "closed"`, `merged == false` |
| Refused signal | Latest review per reviewer is `CHANGES_REQUESTED` (GraphQL `reviewDecision`) | Reviewer `state == "requested_changes"` | Review `state == "REQUEST_CHANGES"`, not dismissed or stale |
| Review summary text | Review `body` | A plain MR note (no review object) | Review `body` |
| Line comments | `GET …/pulls/{n}/comments` | `GET …/merge_requests/:iid/discussions` (`DiffNote`) | `GET …/pulls/{n}/reviews/{id}/comments` |
| Top-level comments | `GET …/issues/{n}/comments` | Same discussions/notes endpoint | `GET …/issues/{n}/comments` |
| Thread resolved flag | GraphQL only (`isResolved`) | REST `notes[].resolved` | `resolver` on review comment |
| Self-review | Author can't approve own PR | Author can't approve by default (Premium setting) | Author can't approve or request changes (422) |

## GitHub

### Publish

- Create: `POST /repos/{owner}/{repo}/pulls` with `head`, `base`, `title`, `body`, and optional `draft` and `maintainer_can_modify`. Returns 201. [GitHub REST: Pulls][gh-pulls]
- Update after rework: push to the same `head` branch. Title and body change with `PATCH /repos/{owner}/{repo}/pulls/{pull_number}`, which also takes `state: open|closed` to reopen or close. [GitHub REST: Pulls][gh-pulls]
- A draft can be marked ready with the GraphQL mutation `markPullRequestReadyForReview` and converted back with `convertPullRequestToDraft`. [GitHub GraphQL schema][gh-schema] The REST `PATCH` endpoint's documented parameters don't include `draft`. I found no REST equivalent, but I didn't exhaustively confirm there isn't one.

A Manifest would hold `number`, `html_url`, `head.sha`, and `base.ref` from the create response.

### Report: detecting Accepted

- `GET /repos/{owner}/{repo}/pulls/{pull_number}` returns `state` (`open`/`closed`), `merged`, `merged_at`, `draft`, `mergeable`, `mergeable_state`, and `auto_merge`. [GitHub REST: Pulls][gh-pulls]
- `GET /repos/{owner}/{repo}/pulls/{pull_number}/merge` returns 204 if merged and 404 if not. [GitHub REST: Pulls][gh-pulls]
- GraphQL `PullRequest.state` is the clearest form. It is one of `OPEN`, `CLOSED` ("closed without being merged"), or `MERGED`. [GitHub GraphQL schema][gh-schema]

Accepted means `merged == true`. `state == "closed"` with `merged == false` is a Shipment closed without merging. That is a third terminal outcome, not a Refusal (see [Mismatches](#where-the-two-operation-shape-strains)).

### Report: detecting Refused

- `GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews` lists every review. Reviews are created with `event` `APPROVE`, `REQUEST_CHANGES`, or `COMMENT`, and `body` is required for the last two. [GitHub REST: Reviews][gh-reviews]
- The REST page doesn't list the `state` values. The GraphQL enum `PullRequestReviewState` does: `APPROVED`, `CHANGES_REQUESTED` ("A review blocking the pull request from merging"), `COMMENTED`, `DISMISSED`, and `PENDING`. [GitHub GraphQL schema][gh-schema]
- The list holds every review ever submitted, so the adapter has to keep only each reviewer's latest non-comment review. GraphQL does this for you:
  - `PullRequest.latestOpinionatedReviews` returns the latest reviews per user and accepts `writersOnly`.
  - `PullRequest.reviewDecision` (`APPROVED`, `CHANGES_REQUESTED`, or `REVIEW_REQUIRED`) gives the overall verdict.

  [GitHub GraphQL schema][gh-schema]
- **Unconfirmed:** whether `reviewDecision` is null when the base branch requires no reviews. The schema marks it nullable but doesn't say when it is null. Treat it as optional and fall back to folding the reviews yourself.
- A requested reviewer drops off `requested_reviewers` once they submit a review. [GitHub REST: Review requests][gh-review-requests]
- Each review carries `commit_id` and `submitted_at` (GraphQL `submittedAt`), which can scope a review to a published head SHA. [GitHub GraphQL schema][gh-schema]

### Feedback location

GitHub keeps feedback in three places:

1. **Review bodies**: the `body` of each review from `…/reviews`. [GitHub REST: Reviews][gh-reviews]
2. **Line comments**: `GET /repos/{owner}/{repo}/pulls/{pull_number}/comments`. Each comment has `path`, `line`, `side`, `start_line`, `commit_id`, `original_commit_id`, `in_reply_to_id`, and `pull_request_review_id`, which links it to its review. `GET …/reviews/{review_id}/comments` scopes the list to one review. The PR-level list doesn't document a `since` filter; the repo-wide list does. [GitHub REST: Review comments][gh-review-comments]
3. **Conversation comments**: a PR is an issue, so top-level comments come from `GET /repos/{owner}/{repo}/issues/{issue_number}/comments`, which supports `since`. [GitHub REST: Issue comments][gh-issue-comments]

REST has no resolved or outdated flag on review comments. [GitHub REST: Review comments][gh-review-comments] GraphQL `PullRequestReviewThread` has `isResolved`, `isOutdated`, `path`, `line`, and `comments`. [GitHub GraphQL schema][gh-schema] To skip feedback that is already resolved, the adapter needs GraphQL.

### GitHub-specific friction

- **Self-review:** "Pull request authors cannot approve their own pull requests." [GitHub Docs: Approving a PR with required reviews][gh-approve] Forgejo's source also blocks request-changes on your own PR. I couldn't find a GitHub doc that says the same for request-changes, so that part is unconfirmed. In both cases Taiichi must publish as an account that no human reviewer uses.
- **Required approvals and stale reviews:** branch protection can require N approvals from writers, dismiss stale approvals on new pushes, require the latest push to be approved by someone other than the pusher, and require conversation resolution. A request-changes review by an admin blocks merging until that person approves, though writers can dismiss it. [GitHub Docs: About protected branches][gh-protected]
- **Auto-merge:** the PR merges itself "after all required reviews and status checks pass". Auto-merge turns off if someone without write access pushes to the head branch. [GitHub Docs: Auto-merge][gh-automerge] The GraphQL mutations are `enablePullRequestAutoMerge` and `disablePullRequestAutoMerge`. [GitHub GraphQL schema][gh-schema]
- **Merge queue:** the PR stays open while queued. If the merge group fails CI, the PR is removed from the queue. [GitHub Docs: Merge queue][gh-mergequeue] GraphQL exposes `isInMergeQueue`, `mergeQueueEntry`, and `MergeQueueEntryState` (`AWAITING_CHECKS`, `LOCKED`, `MERGEABLE`, `QUEUED`, `UNMERGEABLE`). [GitHub GraphQL schema][gh-schema] The synchronous REST merge endpoint "does not support stacked PRs or merge queues", and GitHub recommends `PUT …/merge-async`. [GitHub REST: Pulls][gh-pulls] That only matters if Taiichi ever merges; it doesn't today.

## GitLab

GitLab sources are the Markdown files behind docs.gitlab.com, read from `gitlab-org/gitlab` `master`. Their history notes go up to GitLab 19.5, so some features may be newer than a given self-managed install.

### Publish

- Create: `POST /projects/:id/merge_requests` with `source_branch`, `target_branch`, `title`, and optional `description`, `reviewer_ids`, `remove_source_branch`, `squash`, `target_project_id`, and `allow_collaboration`. [GitLab: Merge requests API][gl-mr]
- **No `draft` parameter.** A Shipment becomes a draft through its title: "Add `[Draft]`, `Draft:` or `(Draft)` to the beginning of the merge request's title." The `/draft` quick action also works. [GitLab: Drafts][gl-drafts] The MR object reports the result as `draft: true`. [GitLab: Merge requests API][gl-mr]

A Manifest would hold the project `id`, the MR `iid` (not the global `id`), `web_url`, and `diff_refs.head_sha`.

### Report: detecting Accepted

- `GET /projects/:id/merge_requests/:merge_request_iid` returns:
  - `state`: one of `opened`, `closed`, `merged`, or `locked`.
  - `merged_at`, `merge_user`, `closed_at`, `closed_by`, `draft`, and `merge_when_pipeline_succeeds` (auto-merge set).
  - `detailed_merge_status`.

  [GitLab: Merge requests API][gl-mr]
- Accepted means `state == "merged"`. `closed` is the closed-unmerged case. `locked` is an extra state that GitHub and Forgejo don't have; the docs list it without explaining it.
- State transitions with timestamps are also available from the resource state events API. [GitLab: Merge requests API][gl-mr]

### Report: detecting Refused

**GitLab does have a "request changes" review state now.** The idea that it lacks one is out of date:

- Reviewers submit a review as **Approve**, **Comment**, or **Request changes**. The reviews page is Free tier. [GitLab: Merge request reviews][gl-reviews]
- In Premium and Ultimate, "A reviewer requesting changes blocks a merge request from merging". This arrived in 16.11 behind a feature flag, was on by default in 17.2, and the flag was removed in 17.3. Another user with merge rights can bypass the block. [GitLab: Merge request reviews][gl-reviews]
- **Polling the verdict:** `GET /projects/:id/merge_requests/:merge_request_iid/reviewers` returns one entry per reviewer with `user`, `state`, and `created_at`. [GitLab: Merge requests API][gl-mr] The REST docs only show `unreviewed` and `reviewed` as examples. The full set, from the model enum in GitLab's source, is:
  - `unreviewed`
  - `reviewed`
  - `requested_changes`
  - `approved`
  - `unapproved`
  - `review_started`

  [GitLab source: `merge_request_reviewer_state.rb`][gl-reviewer-state] The REST entity exposes that same enum attribute. [GitLab source: `merge_request_reviewer.rb`][gl-reviewer-entity] The GraphQL `MergeRequestReviewState` enum matches it. [GitLab GraphQL reference][gl-gql]
- `detailed_merge_status` can also be `requested_changes` ("The merge request has reviewers who have requested changes"), along with `not_approved`, `draft_status`, `discussions_not_resolved`, and others. [GitLab: Merge requests API][gl-mr] **Unconfirmed:** whether this value appears on Free tier, where request-changes doesn't block. Rely on the reviewer state instead.
- Approvals are a separate resource: `GET /projects/:id/merge_requests/:merge_request_iid/approvals` (all tiers) returns `approved` and `approved_by`. On Community Edition, `approved` is true once any approval exists. On Enterprise Edition it is true when the rules are met. [GitLab: MR approvals API][gl-approvals]
- Reviews can be submitted through the API with `POST …/draft_notes/bulk_publish`. Its `reviewer_state` accepts `requested_changes` or `reviewed`, and its `note` adds a summary note. [GitLab: Draft notes API][gl-draft-notes] This is useful for testing the adapter.

### Feedback location

**GitLab's REST API has no review object.** A review submission publishes the reviewer's pending comments, sets their reviewer state, and can add a summary note. [GitLab: Merge request reviews][gl-reviews] [GitLab: Draft notes API][gl-draft-notes] Nothing in REST links those notes back to one review submission. The adapter has to rebuild "the feedback for this Refusal" from:

- `GET /projects/:id/merge_requests/:merge_request_iid/discussions`. This returns threads, each with `notes[]` carrying:
  - `type`: `DiffNote` for a line comment, `DiscussionNote`, or `null` for a plain note.
  - `body`, `author`, `created_at`, `system`, `resolvable`, `resolved`, `resolved_by`, and `position` (file and line for diff notes).

  [GitLab: Discussions API][gl-discussions]
- `GET /projects/:id/merge_requests/:merge_request_iid/notes` (sort by `created_at` or `updated_at`). This is the flat list of top-level notes. [GitLab: Notes API][gl-notes]
- System notes (`system: true`) must be filtered out. [GitLab: Discussions API][gl-discussions]

The practical rule is: collect every non-system note by the refusing reviewer created after the last Publish, plus every unresolved resolvable thread.

### GitLab-specific friction

- **Self-approval:** "By default, the creator of a merge request (author) cannot approve it". Premium can change this, and there is also a setting that blocks approvals from users who added commits. [GitLab: Approval settings][gl-approval-settings] This page is Premium-tier, so I couldn't confirm the Free-tier rule. I also found no rule that stops an author from requesting changes on their own MR. Either way, a separate bot account is required.
- **Auto-merge and merge trains:** `PUT …/merge` with `auto_merge: true` merges "when checks pass". On projects with merge trains, the API adds the MR to the train instead (since 19.1). [GitLab: Merge requests API][gl-mr] If a train pipeline fails, the MR "is dropped from the train" and isn't merged. [GitLab: Merge trains][gl-trains] The MR stays `opened` throughout, so polling only sees `merged` when it actually lands.
- **Change requests persist.** A reviewer clears a change request by re-reviewing and approving, or by removing it. Someone with merge rights can also bypass it. [GitLab: Merge request reviews][gl-reviews] The author can re-request review, which shows in the UI and through the `/request_review` quick action. [GitLab: Merge request reviews][gl-reviews] GraphQL has the `mergeRequestReviewerRereview` mutation. [GitLab GraphQL reference][gl-gql]
  - **Unconfirmed:** whether a push resets `requested_changes` to `unreviewed`.
  - The REST reviewer entity has no timestamp for when the state last changed; `created_at` is when the reviewer was added. So the adapter can't tell a stale change request from a fresh one by reviewer state alone. It has to correlate with note timestamps, or re-request review on each Publish so a new verdict is distinguishable.

## Forgejo (v1 baseline, for comparison)

Sources: Codeberg's live OpenAPI spec (`info.version` `16.0.0-dev-785-465b60ef+gitea-1.22.0`) and Forgejo's source on the `forgejo` branch.

- **Publish:** `POST /repos/{owner}/{repo}/pulls` with `head`, `base`, `title`, `body`, and others. There is no `draft` field. [Forgejo OpenAPI][fj-swagger] Work-in-progress status comes from a title prefix, `WIP:` or `[WIP]` by default, set by `WORK_IN_PROGRESS_PREFIXES`. [Forgejo config cheat sheet][fj-config]
- **Accepted:** `GET /repos/{owner}/{repo}/pulls/{index}` returns `merged`, `merged_at`, `merged_by`, `merge_commit_sha`, `state`, `closed_at`, `draft`, and `requested_reviewers`. `GET …/pulls/{index}/merge` also exists. [Forgejo OpenAPI][fj-swagger]
- **Refused:** `GET …/pulls/{index}/reviews` returns `PullReview` objects with `state`, `body`, `commit_id`, `stale`, `official`, `dismissed`, and `submitted_at`. [Forgejo OpenAPI][fj-swagger] The spec leaves `ReviewStateType` without enum values. The source defines `APPROVED`, `PENDING`, `COMMENT`, `REQUEST_CHANGES`, `REQUEST_REVIEW`, and `""`. [Forgejo source: `pull_review.go`][fj-review-structs]
- **Feedback:**
  - Review bodies.
  - Line comments from `GET …/pulls/{index}/reviews/{id}/comments`. Each has `path`, `position`, `original_position`, `diff_hunk`, `commit_id`, and a `resolver` that is set once resolved.
  - Top-level comments from the issue comments endpoint.

  [Forgejo OpenAPI][fj-swagger] Unlike GitHub, there's no PR-wide list of line comments; the adapter walks each review.
- **Self-review:** the API returns 422 "approve your own pull is not allowed" and "reject your own pull is not allowed". [Forgejo source: `routers/api/v1/repo/pull_review.go`][fj-review-router]
- **Protection and merging:** branch protection has `required_approvals`, `block_on_rejected_reviews`, `block_on_official_review_requests`, `dismiss_stale_approvals`, and `ignore_stale_approvals`. The merge form supports `merge_when_checks_succeed` (auto-merge) and a `manually-merged` mode. [Forgejo OpenAPI][fj-swagger]

Forgejo's model is close to GitHub's: reviews with bodies and states, line comments that belong to a review, and issue comments for the conversation. Its per-review `stale` flag makes "is this Refusal about the current head?" easier than on GitHub or GitLab.

## Where the two-operation shape strains

1. **Closed without merging.** All three Forges have it: GitHub `CLOSED`, GitLab `closed`, Forgejo `state: closed` with `merged: false`. It is neither Accepted nor Refused, and it usually carries no feedback. Mapping it to Refused would send Build into a rework loop on a Shipment no one will review. Report needs a separate terminal outcome, *Closed* or *Abandoned*, that surfaces the Job to a human. Reopening is possible on all three, so polling a closed Shipment once more before giving up is cheap.

2. **Approved but not merged.** Approval doesn't mean merged on any Forge. Required approvals, required checks, auto-merge, GitHub merge queues, and GitLab merge trains all leave the Shipment open after approval. [GitHub: protected branches][gh-protected] [GitHub: merge queue][gh-mergequeue] [GitLab: merge trains][gl-trains] Accepted should strictly mean merged. "Approved, waiting to merge" is *Pending*. A failed merge-queue or train run drops the Shipment back to open with no new feedback. Taiichi can't act on that through the Report operation; it is a CI failure, and the Shipment just stays Pending.

3. **Pending isn't in the interface.** Polling needs a "no verdict yet" result. It is implied, but the interface should say so.

4. **Change requests persist across pushes.** GitHub keeps a reviewer's `CHANGES_REQUESTED` until they review again or someone dismisses it. GitLab keeps `requested_changes` until the reviewer approves, removes it, or someone bypasses it. Forgejo marks reviews `stale`. [GitHub: protected branches][gh-protected] [GitLab: reviews][gl-reviews] [Forgejo OpenAPI][fj-swagger] After Build reworks and re-Publishes, a naive poll reports the old Refusal again. To prevent that:
   - Store the published head SHA and a feedback cursor in the Manifest.
   - Report Refused only for a change request submitted against, or after, the latest published head. Use GitHub `commit_id`/`submitted_at`, Forgejo `commit_id`/`stale`, or GitLab note timestamps plus a re-request.
   - Optionally re-request review from the refusing reviewers on re-Publish. GitHub has `POST …/requested_reviewers/rerequest`, GitLab has `mergeRequestReviewerRereview`, and Forgejo has `POST …/requested_reviewers`. [GitHub REST: Review requests][gh-review-requests] [GitLab GraphQL reference][gl-gql] [Forgejo OpenAPI][fj-swagger]

5. **Publish is really "publish or update".** The Refusal → Build → Publish loop should push to the same branch and keep the same Shipment, so reviewers see a diff. Opening a new PR or MR each round throws the discussion away. Publish must be idempotent per Job: find or create the Shipment, push, and return an updated Manifest.

6. **Comment-only feedback.** All three Forges let a reviewer leave comments without approving or requesting changes (GitHub `COMMENTED`, GitLab `reviewed`, Forgejo `COMMENT`). The interface needs a policy for this. Recommendation: stay *Pending* and don't trigger Build. Otherwise drive-by comments start rework.

7. **Drafts.** GitHub has a real `draft` flag. GitLab and Forgejo use title prefixes. [GitHub REST: Pulls][gh-pulls] [GitLab: Drafts][gl-drafts] [Forgejo config][fj-config] If Taiichi ever publishes drafts, the adapter must own the title prefix on GitLab and Forgejo. The prefix on Forgejo is per instance (configurable).

8. **Self-review and identity.** GitHub (approve), GitLab (approve, by default), and Forgejo (approve and request changes) all block the author from giving a verdict on their own Shipment. [GitHub: approving][gh-approve] [GitLab: approval settings][gl-approval-settings] [Forgejo source][fj-review-router] Taiichi's Forge credential must be a bot account separate from every reviewer. Otherwise the human can't Refuse or Accept through the review UI, and no verdict reaches the poll. The design should make this explicit.

9. **The shape of feedback.** No Forge offers "the feedback" as one resource. Each needs two or three calls: review bodies, line comments or threads, and top-level comments. GitLab also needs reconstruction because it has no review object. The adapter should return a normalised list of feedback items: author, body, optional path and line, resolved flag, and timestamp. Its Refused payload shouldn't be a single string.

10. **Verdicts from several reviewers.** With many reviewers, each Forge's verdict is a fold over the latest opinion per reviewer. GitHub's `reviewDecision` is the only built-in fold. "Refused" should mean that at least one current reviewer has an outstanding change request, matching how GitHub and GitLab block merges.

## Assessment

**The interface holds.** All three Forges support "open a Shipment" and "poll for a merged or changes-requested verdict with feedback" with plain REST calls (GraphQL is optional on GitHub). GitLab's "Request changes" state exists on all tiers and blocks merging on Premium and above. It is the per-reviewer state `requested_changes`, not a review object, which makes the GitLab adapter the most work. No Forge needs a third operation for the factory's flow.

Recommended changes:

1. **Widen Report's result** from `Accepted | Refused(feedback)` to `Pending | Accepted | Refused(feedback) | Closed`. Accepted strictly means merged. Approved-but-unmerged, queued, or on a train is Pending.
2. **Make Publish idempotent per Job** ("publish or update"). It reuses the Shipment and branch on rework and returns a fresh Manifest.
3. **Add to the Manifest:** the Forge-native Shipment id (GitHub/Forgejo `number`, GitLab project `id` and `iid`), the URL, the published head SHA, and a feedback cursor (a timestamp or last-seen IDs). Report uses them to ignore verdicts and comments from earlier rounds.
4. **Define feedback as a list of structured items** (author, body, optional file and line, resolved flag, created_at). It should not be a single blob.
5. **Write down two policies:** comment-only reviews leave the Shipment Pending, and the Forge identity must be a dedicated bot account that never reviews.
6. Optionally have Publish **re-request review** from earlier refusers on rework. This makes fresh verdicts unambiguous, which matters most on GitLab, where reviewer state has no change timestamp.

## Not confirmed

- When GitHub's `reviewDecision` is null, for example on branches that require no reviews.
- Whether GitHub blocks an author from submitting `REQUEST_CHANGES` on their own PR. Only the approval restriction is documented.
- Whether GitHub REST can switch a PR between draft and ready. Only GraphQL mutations were found.
- Whether a push to a GitLab MR resets a reviewer's `requested_changes` state.
- Whether GitLab's `detailed_merge_status: requested_changes` appears on Free tier.
- GitLab's Free-tier rule for author self-approval. The approval settings page is Premium-only.
- What GitLab's `locked` MR state means in practice.
- Forgejo was checked against Codeberg's dev-version spec (16.0.0-dev). Field names may differ slightly on older releases.

[gh-pulls]: https://docs.github.com/en/rest/pulls/pulls
[gh-reviews]: https://docs.github.com/en/rest/pulls/reviews
[gh-review-comments]: https://docs.github.com/en/rest/pulls/comments
[gh-review-requests]: https://docs.github.com/en/rest/pulls/review-requests
[gh-issue-comments]: https://docs.github.com/en/rest/issues/comments
[gh-schema]: https://docs.github.com/public/fpt/schema.docs.graphql
[gh-approve]: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/approving-a-pull-request-with-required-reviews
[gh-protected]: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches
[gh-automerge]: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/incorporating-changes-from-a-pull-request/automatically-merging-a-pull-request
[gh-mergequeue]: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue
[gl-mr]: https://docs.gitlab.com/api/merge_requests/
[gl-approvals]: https://docs.gitlab.com/api/merge_request_approvals/
[gl-approval-settings]: https://docs.gitlab.com/user/project/merge_requests/approvals/settings/
[gl-discussions]: https://docs.gitlab.com/api/discussions/
[gl-notes]: https://docs.gitlab.com/api/notes/
[gl-draft-notes]: https://docs.gitlab.com/api/draft_notes/
[gl-reviews]: https://docs.gitlab.com/user/project/merge_requests/reviews/
[gl-drafts]: https://docs.gitlab.com/user/project/merge_requests/drafts/
[gl-trains]: https://docs.gitlab.com/ci/pipelines/merge_trains/
[gl-gql]: https://docs.gitlab.com/api/graphql/reference/
[gl-reviewer-state]: https://gitlab.com/gitlab-org/gitlab/-/blob/master/app/models/concerns/merge_request_reviewer_state.rb
[gl-reviewer-entity]: https://gitlab.com/gitlab-org/gitlab/-/blob/master/lib/api/entities/merge_request_reviewer.rb
[fj-swagger]: https://codeberg.org/swagger.v1.json
[fj-config]: https://forgejo.org/docs/latest/admin/config-cheat-sheet/
[fj-review-structs]: https://codeberg.org/forgejo/forgejo/src/branch/forgejo/modules/structs/pull_review.go
[fj-review-router]: https://codeberg.org/forgejo/forgejo/src/branch/forgejo/routers/api/v1/repo/pull_review.go
