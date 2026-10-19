# A Shipment changing under a Job

Type: grilling
Status: open

## Question

What happens when a Job's Shipment changes while the Job is away from Ship, in Build or Evaluate?

- The pull request is merged: the Pipeline reaches Done only from Ship, and Report only runs at Ship.
- The pull request is closed.
- Someone else pushes to the branch: the worktree falls behind, the next Build's starting commit is off, and Ship's `--force-with-lease` refuses to overwrite the push ([Forgejo Forge](08-forgejo-forge.md)).

For each case: does the server notice (the poller runs only while a Job waits at Ship, per [Server internals](12-server-internals.md)), what does the Job do, and what does the human see?

## Comments

**From [An operation stopping partway](15-operation-stopping-partway.md):** Refused now counts any request-changes review newer than the Ship marker, whatever commit it names, so a head someone else moved doesn't hide a refusal. What the Job does when the head moves is still for this ticket. Abandon at Ship already checks for a merge first; merges while the Job is away from Ship are still here.
