# Server internals

Type: grilling
Status: open
Blocked by: 05

## Question

How is the server built inside?

- The process model for Jobs (for example, one `gen_statem` per Job enforcing the Pipeline) and for polling.
- The on-disk layout of a Job's data folder: a folder per Attempt, numbered within its Stage, holding the Stage Input bundle and the Stage Output (see [Stage Inputs and Stage Outputs](05-stage-inputs-and-outputs.md)).
- How state is rebuilt from disk after a restart, and how writes stay safe if the server stops mid-write.
- How "one active Job per Ticket" is enforced.
