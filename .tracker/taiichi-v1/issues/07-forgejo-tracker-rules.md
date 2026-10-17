# Forgejo Tracker rules

Type: grilling
Status: open
Blocked by: 02

## Question

How does the Forgejo Tracker behave, and what can each project configure?

- Which issues are eligible for Pickup (for example, open and labelled `ready-for-human` or `ready-for-agent`)?
- What the Ticket id looks like on the CLI, and how branch names are built from it.
- How each Job position maps to an issue change (labels, assignee, comments, closing), and how the issue closes when the Shipment is Accepted.
- The per-project knobs for that mapping, since two projects on the same Tracker can map differently.

Build on the findings of [Forgejo API for the Tracker and Forge](02-forgejo-api.md).

Facts from [Forgejo API for the Tracker and Forge](02-forgejo-api.md) that bear on this: label filters match *all* listed labels and silently drop unknown names; exclusive (scoped) labels replace others in the same scope, which may suit Job-position labels; assignees can only be replaced as a whole set.
