# Taiichi

A software factory that pulls Tickets from a Tracker and tracks each one as it moves through a fixed Pipeline of Stages.

## Language

### Work

**Repo**:
A git repository that has opted in with `taiichi init`. It names its Tracker and Forge.
_Avoid_: Project

**Tracker**:
The project management system Tickets come from, such as Forgejo, GitHub Issues, or Linear.
_Avoid_: PM system, source

**Ticket**:
A work item as it lives in a Tracker. The factory reads it but does not own it.
_Avoid_: Issue, task, story

**Job**:
The factory's own record of one Ticket moving through the Pipeline, including its current Position and its history. Each Job has a number that is unique across the server and never reused.
_Avoid_: Ticket (for the factory's record), run, work order

**Pickup**:
The act of starting a Job from a Ticket.

**Forge**:
The place a Job's work is shipped for review and acceptance, such as Forgejo, GitHub, or GitLab.
_Avoid_: Host, remote

**Shipment**:
The work a Forge holds for review, such as a pull request on Forgejo or GitHub. Every Ship of a Job updates the same Shipment, and a Report on it covers only the latest Ship.
_Avoid_: Pull request, PR, merge request

**Foreign commit**:
A commit on a Job's branch that someone other than the factory put on the Forge, such as a fix a reviewer pushed. It is part of the Job's work, and the next Ship must include it.

**Feedback**:
The comments left on a Shipment since the last Feedback, gathered when the Shipment is Refused and carried back to Build in its Stage Input. Anyone but the factory's own account may contribute.
_Avoid_: Review comments, change requests

**Report**:
What an outside system says about something a Job is waiting on, each time it is checked. A Report's value means something only alongside what it reports on, so values are grouped by subject below.
_Avoid_: Outcome, status

### Reports on a Shipment

**Pending**:
The Shipment is still under review, including one approved but not yet taken.

**Accepted**:
The reviewer took the Shipment, which moves the Job to Done.
_Avoid_: Merged, approved

**Refused**:
The reviewer sent the Shipment back with changes requested, which returns the Job to Build.
_Avoid_: Rejected, changes requested

**Closed**:
The Shipment was withdrawn from review without being Accepted, such as a pull request closed without merging. The Job can't Ship again until a human reopens the Shipment, and a human may abandon the Job instead.
_Avoid_: Abandoned (for the Shipment), declined

### Pipeline

**Pipeline**:
The fixed state machine every Job moves through: Plan, Build, Evaluate, Ship, with loops back to Build, ending in Done or Abandoned from any Stage.
_Avoid_: Workflow, process

**Stage**:
A position in the Pipeline where work happens: Plan, Build, Evaluate, or Ship.
_Avoid_: Step, phase

**Position**:
Where a Job stands in the Pipeline: one of the four Stages, at an Attempt, or one of the two ends, Done or Abandoned.
_Avoid_: State, status

**Attempt**:
One try of a Job at a Stage, numbered within that Stage (Build 1, Build 2) so that repeated Attempts are kept apart. Its Output may be sent back, which starts the next Attempt.
_Avoid_: Pass, iteration, run

**Done**:
The end of the Pipeline for a Job, reached when the Job's Shipment is Accepted, from whichever Stage the Job is in.

**Abandoned**:
The end of the Pipeline for a Job that was dropped before being accepted.
_Avoid_: Cancelled, closed

### Stage data

**Stage Input**:
Everything a Stage receives when a Job enters it: new material such as a Ticket Snapshot or Feedback, plus references to earlier Stage Outputs. Shortened to Input after a possessive that already names the Stage or Attempt, as in "Build's Input".

**Stage Output**:
The structured data a Stage produces for a Job before the Job leaves it. Shortened to Output after a possessive that already names the Stage or Attempt, as in "Plan's Output".

**Ticket Snapshot**:
The copy of a Ticket taken once at Pickup, which Plan works from and which never changes afterward.
_Avoid_: Ticket copy

**Blueprint**:
The Stage Output of Plan: how the Ticket will be built.
_Avoid_: Plan (for the output)

**Build Notes**:
The Stage Output of Build: what was built and how.

**Evaluation**:
The Stage Output of Evaluate, carrying a Verdict.

**Verdict**:
The pass-or-fail judgment in an Evaluation that sends the Job on to Ship or back to Build.

**Manifest**:
The Stage Output of Ship: a record of what the Shipment contains and where it lives.
