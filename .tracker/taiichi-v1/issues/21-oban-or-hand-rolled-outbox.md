# Oban or the hand-rolled outbox

Type: grilling
Status: resolved
Blocked by: 20

## Question

Given [Does Oban fit the outbox and the pollers?](20-oban-fit.md), should the server use Oban for the effects after a save, and possibly for the Ticket and Shipment checks, in place of the `outbox` table and sender settled in [An operation stopping partway](15-operation-stopping-partway.md)? If so, which settled rules change (ordering, giving up, retrying at startup, keeping finished rows, the Job object's `outbox` field), and what does [Server internals](12-server-internals.md) lose or gain?

## Answer

Settled with the user. Glossary: no change. No ADR: the outbox is internal to the server, nothing outside Taiichi reads it, and moving its rows into `oban_jobs` later would be one migration, so the choice is reversible. The reasons are recorded here.

**The outbox stays hand-rolled**
- The `outbox` table and its one sender process stay exactly as settled in [An operation stopping partway](15-operation-stopping-partway.md). No settled rule changes: ordering per Job and place, retries every `poll_interval`, the 24-hour give-up, retrying given-up rows at start, keeping finished rows, `last_error`, and the Job object's `outbox` field.
- **Open-source Oban was weighed and rejected.** Of the outbox's eight rules, it meets three as it ships: inserting in the save's transaction, keeping finished jobs, and a fixed retry interval. Taiichi would still write code for the other five: the ordering gate, the 24-hour time check, restarting the window at startup, resetting orphaned `executing` jobs, and waiting for the first try through telemetry. Deleting a Job would also need an `args` query, since there's no foreign key. Oban would add a table, a supervisor tree and a database query every second. It would also change two settled behaviours:
  - The first try starts on the stager's one-second tick, so `pickup`, `submit` and `abandon` replies would wait up to a second longer.
  - Each failure adds an entry to the job's `errors` list, 1,440 a day at 60 seconds, where the outbox keeps only `last_error`.
- **Oban Pro was rejected.** Its chains match the ordering rule, but no first-party source says they run on SQLite. Its licence is proprietary, forbids modification and third-party access, and carries a two-year non-compete.
- The worker code that asks git or Forgejo whether a step is done, and skips duplicate comments, is Taiichi's either way, so it didn't count toward either side.

**The checks stay on the checker's timer**
- Each active Job's checker keeps ticking every `poll_interval` with `Process.send_after/3`, as settled in [A Ticket changing under a running Job](17-ticket-changing-under-a-job.md). That works at any interval and writes no rows.
- Oban cron was rejected: it has one-minute resolution, so it can only express a `poll_interval` that `*/n` can (not 30 or 90 seconds), and its crontab is fixed at boot.
- Self-rescheduling Oban jobs were rejected: they work at any interval, but write a row per Job per tick, and with no pruner those rows are kept forever.

**Changes to earlier decisions**
- None. [Server internals](12-server-internals.md) neither loses nor gains anything.
- [Assemble the v1 spec](13-assemble-the-v1-spec.md): the spec should say in one line that the outbox is hand-rolled rather than built on Oban, linking here, so no one reopens it while slicing the work.

Findings: `docs/research/oban-fit.md` on branch `research/oban-fit`, commit `fea32df`.
