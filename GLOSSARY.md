# Taiichi

A software factory that pulls Tickets from a Tracker and tracks each one as it moves through a fixed Pipeline of Stages.

## Language

### Work

**Tracker**:
The project management system Tickets come from, such as Forgejo, GitHub Issues, or Linear.
_Avoid_: PM system, source

**Ticket**:
A work item as it lives in a Tracker. The factory reads it but does not own it.
_Avoid_: Issue, task, story

**Job**:
The factory's own record of one Ticket moving through the Pipeline, including its current position and its history.
_Avoid_: Ticket (for the factory's record), run, work order

**Pickup**:
The act of starting a Job from a Ticket.

**Forge**:
The place a Job's work is shipped for review and acceptance, such as Forgejo, GitHub, or GitLab.
_Avoid_: Host, remote

**Shipment**:
The work a Forge holds for review until it is Accepted or Refused, such as a pull request on Forgejo or GitHub.
_Avoid_: Pull request, PR, merge request

**Accepted**:
The outcome of a Shipment that the reviewer takes, which moves the Job to Done.
_Avoid_: Merged, approved

**Refused**:
The outcome of a Shipment that the reviewer sends back with changes requested, which returns the Job to Build.
_Avoid_: Rejected, changes requested

**Feedback**:
The reviewer's list of comments on a Refused Shipment, carried back to Build in its Stage Input.
_Avoid_: Review comments, change requests

### Pipeline

**Pipeline**:
The fixed state machine every Job moves through: Plan, Build, Evaluate, Ship, with loops back to Build, ending in Done or Abandoned.
_Avoid_: Workflow, process

**Stage**:
A position in the Pipeline where work happens: Plan, Build, Evaluate, or Ship.
_Avoid_: Step, phase

**Attempt**:
One entry of a Job into a Stage, numbered within that Stage (Build 1, Build 2) so that repeated visits are kept apart.
_Avoid_: Pass, iteration, run

**Done**:
The end of the Pipeline for a Job whose Shipment was Accepted.

**Abandoned**:
The end of the Pipeline for a Job that was dropped before being accepted.
_Avoid_: Cancelled, closed

### Stage data

**Stage Input**:
Everything a Stage receives when a Job enters it: new material such as a Ticket Snapshot or Feedback, plus references to earlier Stage Outputs.

**Stage Output**:
The structured data a Stage produces for a Job before the Job leaves it.

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
