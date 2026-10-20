# Assemble the v1 spec

Type: task
Status: resolved
Blocked by: 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 14, 15, 16, 17, 18, 19, 20, 21

## Question

Assemble the decisions recorded across this map into a single v1 spec, ready to hand off and slice into implementation issues. Nothing new is decided here; any gap found goes back to the map as a ticket.

## Answer

Assembled as [the v1 spec](../spec.md), from every resolved ticket on the map, `GLOSSARY.md`, and [SQLite for the server's data](../../../docs/adr/0001-sqlite-for-server-data.md). Where tickets disagreed, the later decision won, following the "Changes to earlier decisions" sections and the comments each ticket left on earlier ones. The spec is arranged for slicing: Pipeline, Stage data, operations, checker, outbox, each adapter, configuration, storage, server structure, API, and CLI, each section linking its source tickets. It says in one line that the outbox is hand-rolled rather than built on Oban, linking [Oban or the hand-rolled outbox](21-oban-or-hand-rolled-outbox.md), as that ticket asked.

**Gaps went back to the map** as one ticket, [Gaps found assembling the v1 spec](22-gaps-found-assembling-the-spec.md), with five questions: reading the Ticket Snapshot and Feedback after `input` moves on; the Job number in Pickup's branch check; removing a worktree kept at Done; whether `status` returns the Forgejo account; and how `init`'s exclusive-scope prompt travels over the API. Each is marked **Open** in the spec, and that ticket updates the spec when it closes.

**Readings:** nine places where the tickets imply an answer without stating it are written into the spec as read, and listed in its last section for the user to correct. Among them: the worktree path is stored at Pickup's save, Refused moves only a Job waiting on its Shipment, and the two unnamed Shipment fields on the Job object are called `worktree_behind` and `new_comments`.

No glossary change and no ADR: nothing was decided.

## Comments

**From [Oban or the hand-rolled outbox](21-oban-or-hand-rolled-outbox.md):**
- Say in one line that the outbox is hand-rolled rather than built on Oban, linking that ticket, so no one reopens it while slicing the work.
