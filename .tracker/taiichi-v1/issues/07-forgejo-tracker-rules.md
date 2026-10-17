# Forgejo Tracker rules

Type: grilling
Status: resolved
Blocked by: 02

## Question

How does the Forgejo Tracker behave, and what can each project configure?

- Which issues are eligible for Pickup (for example, open and labelled `ready-for-human` or `ready-for-agent`)?
- What the Ticket id looks like on the CLI, and how branch names are built from it.
- How each Job position maps to an issue change (labels, assignee, comments, closing), and how the issue closes when the Shipment is Accepted.
- The per-project knobs for that mapping, since two projects on the same Tracker can map differently.

Build on the findings of [Forgejo API for the Tracker and Forge](02-forgejo-api.md).

Facts from [Forgejo API for the Tracker and Forge](02-forgejo-api.md) that bear on this: label filters match *all* listed labels and silently drop unknown names; exclusive (scoped) labels replace others in the same scope, which may suit Job-position labels; assignees can only be replaced as a whole set.

## Answer

Settled with the user. No glossary change: ready labels and position labels are settings of the Forgejo adapter, not domain terms.

**Eligibility and Pickup**
- An issue is eligible for Pickup when it is open, is not a pull request, carries at least one of the project's ready labels, and has no active Job. Forgejo's label filter matches *all* listed labels, so the adapter queries once per ready label and merges the results.
- `taiichi pickup` refuses an ineligible Ticket and says why (closed, a pull request, missing a ready label, or already has an active Job).
- The server checks that every configured label (ready and position) exists in the repo and fails clearly if one doesn't, because Forgejo silently ignores unknown label names. The server never creates labels; that is left to [`taiichi init` and configuration](09-init-and-configuration.md).

**Ids and branches**
- A Forgejo Ticket id is the bare issue number (`42`), and the adapter rejects any other form. `#42` is refused because the shell treats a word starting with `#` as a comment. The core treats Ticket ids as opaque strings, so other Trackers can use forms like `ENG-123`.
- The Tracker's config names its own `owner/repo`, defaulting to the Forge's repo.
- Branches are named `taiichi/<ticket>-j<job>-<slug>`, for example `taiichi/42-j7-add-login-page`. Including the Job number keeps names unique, since branches outlive Abandoned Jobs. The slug comes from the Ticket Snapshot's title: lowercased, with anything that isn't a letter or digit turned into a hyphen, and trimmed to a fixed length.

**Write-back**
- **Position labels:** each project maps each position (Plan, Build, Evaluate, Ship, Done, Abandoned) to a label name, or to nothing. The defaults are `taiichi/plan` through `taiichi/abandoned`. On each change, the adapter removes the previous position label and adds the new one, without relying on Forgejo's exclusive scopes, so any names work; exclusive scoped labels are still the suggested setup. At Pickup it clears any position label left by an earlier Job.
- **Ready labels:** Pickup removes every ready label the issue carries, so Forgejo's own issue list tells waiting issues apart from in-progress ones. On Abandoned, the adapter restores the ready labels it removed, read from the Ticket Snapshot (taken before the removal), so the issue is eligible again and keeps `taiichi/abandoned` as a trace. On Done they stay off, since the issue closes.
- **Comments:** one on Pickup (Job number and branch), one on Done (the Accepted Shipment's URL), and one on Abandoned (the branch is kept). Each can be switched off per project. No comments on the changes in between, since the position label shows them.
- **No assignee changes:** Taiichi doesn't know which human is doing a Stage, and Forgejo can only replace the whole assignee set.
- **Closing:** when the Job reaches Done, the Tracker closes the issue directly. The Forge puts no closing keyword in the pull request, which keeps the two adapters independent. Whether the pull request *mentions* the issue is left to [Forgejo Forge](08-forgejo-forge.md). An Abandoned Job leaves its issue open.

**Per-project knobs:** the Tracker's `owner/repo`, the ready-label list, the position-label map, and the three comment switches.

## Comments

**From [`taiichi init` and configuration](09-init-and-configuration.md):** two rules here changed.
- The `abandoned` position label is dropped, so the position map covers Plan, Build, Evaluate, Ship, and Done. Abandon removes the current position label and restores the ready labels, with no trace label left on the issue. The Abandoned comment and the Job history still record it.
- The default ready label is `taiichi/ready`, created by `taiichi init` as an exclusive label in the `taiichi` scope next to the position labels. "Per-project" settings are now per Repo.
