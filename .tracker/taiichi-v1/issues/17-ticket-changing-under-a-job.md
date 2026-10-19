# A Ticket changing under a running Job

Type: grilling
Status: open

## Question

What happens when a Job's Ticket changes on the Tracker while the Job is active?

- The Ticket is edited: the Ticket Snapshot never changes, so is the edit ignored, reported, or a reason to stop?
- The Ticket is closed.
- Someone relabels it, including re-adding a ready label that Pickup removed, or removing or swapping a position label.
- Taiichi's own writes coming back: it swaps labels on every move and closes the Ticket on Done, so a Report on a Ticket could be its own change. Feedback faces the same problem and leaves out the factory's own account. If the Tracker reports on Tickets, its values get their own group in `GLOSSARY.md`, `### Reports on a Ticket`, beside `### Reports on a Shipment`.

Does the server ever look at the Ticket after Pickup? [Forgejo Tracker rules](07-forgejo-tracker-rules.md) settled what the Tracker writes; this settles what it does about changes it didn't make.
