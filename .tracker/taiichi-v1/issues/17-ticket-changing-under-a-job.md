# A Ticket changing under a running Job

Type: grilling
Status: open

## Question

What happens when a Job's Ticket changes on the Tracker while the Job is active?

- The Ticket is edited: the Ticket Snapshot never changes, so is the edit ignored, reported, or a reason to stop?
- The Ticket is closed.
- Someone relabels it, including re-adding a ready label that Pickup removed, or removing or swapping a position label.

Does the server ever look at the Ticket after Pickup? [Forgejo Tracker rules](07-forgejo-tracker-rules.md) settled what the Tracker writes; this settles what it does about changes it didn't make.
