# Gates

Status: needs-triage

Once a second operator drives Stages (for example, an AI agent), the primary operator may want to check each Stage Output before the Job moves on, and send it back with notes. A sketch:

1. A Ticket is picked up.
2. Plan writes a Blueprint.
3. The Plan gate sends the Blueprint back with notes, before the Job enters Build.
4. Plan runs again, as Plan 2.

A gate is named for the Stage whose work it checks: it reads that Stage's Output, and sending it back repeats that Stage. So step 3 is the Plan gate, not the Build gate.

v1 already allows this: Attempts are numbered within each Stage, so a Plan 2 needs no change to how they're stored. See [An operation stopping partway](../../taiichi-v1/issues/15-operation-stopping-partway.md).

What to settle then:
- **Evaluate and Ship already judge Build's work** and send it back with comments. Is a Build gate a separate human check, or is Evaluate that gate?
- **Feedback** is defined as comments on a Shipment. A gate's notes either widen that term or need their own.
- **Position** gains a new wait: Stage Output submitted, awaiting the gate.
- **The Stage Input** of a repeated Stage: the gate's notes, and which earlier Stage Outputs.
- **The fixed Pipeline** loops back only to Build today. Gates add a loop from every Stage back to itself.
- Whether gates are set per Repo or per Stage, and who may pass one.
