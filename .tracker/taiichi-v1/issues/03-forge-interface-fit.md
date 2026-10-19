# Does the Forge interface fit GitHub and GitLab?

Type: research
Status: resolved

## Question

Taiichi's Forge adapter has two operations (see [Chart the v1 scope](01-chart-the-v1-scope.md)):

1. **Publish** a Job's branch as a Shipment (a pull request or merge request), producing a Manifest.
2. **Report** on the Shipment by polling: Accepted (merged) or Refused (changes requested), with the reviewer's feedback.

Do GitHub's and GitLab's APIs map cleanly onto these two operations, alongside Forgejo (which Codeberg runs)? For each, identify the calls behind each operation, how "Accepted" and "Refused" are detected, where the reviewer's feedback lives, and any place the two-operation shape doesn't fit (for example, a Shipment closed without merging, approvals without merging, or draft states).

Primary sources: GitHub's REST API docs and GitLab's REST API docs.

## Answer

**The two-operation interface holds** for GitHub, GitLab and Forgejo. Each has one call to open a Shipment, a merged flag for Accepted, and a request-changes review state for Refused (GitLab gained one in 16.11). It needs four adjustments:

- **A Report can say more:** Pending, Accepted, Refused (with feedback), or Closed. A Shipment closed without merging is not a Refusal. Accepted strictly means merged, so approved-but-unmerged is Pending.
- **Report is scoped to the latest Publish:** change requests stay in force after new pushes on GitHub and GitLab, so the Manifest needs the published head commit and a feedback cursor to avoid re-reporting an old Refusal.
- **Publish is "publish or update":** rework pushes to the same branch and Shipment rather than opening a new one.
- **Feedback is a list of structured items**, gathered from two or three API resources on every Forge. GitLab is the hardest, since it has no review object in its REST API.

Other constraints: all three Forges stop an author from approving their own Shipment, and Forgejo rejects an author requesting changes on their own pull request. So if Taiichi publishes as the same account a human reviews with, that reviewer's verdict can't be recorded. Comment-only reviews and drafts need a policy.

Findings: `docs/research/forge-interface-fit.md` on branch `research/forge-interface-fit` (commit `84d16c7`). The branch is local only; the push failed because the sandbox has no `ssh`.
