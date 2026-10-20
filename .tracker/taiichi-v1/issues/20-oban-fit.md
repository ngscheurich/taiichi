# Does Oban fit the outbox and the pollers?

Type: research
Status: resolved

## Question

[An operation stopping partway](15-operation-stopping-partway.md) settled a hand-rolled `outbox` table and sender. The user wants [Oban](https://oban.hexdocs.pm/Oban.html) considered in its place. Which of the outbox's settled rules can open-source Oban meet, and how?

- Whether open-source Oban runs on SQLite (the Lite engine), with the connection settings from [Server internals](12-server-internals.md) (WAL, busy timeout, `synchronous = NORMAL`).
- Inserting a job in the same Ecto transaction as the save (`Ecto.Multi` or `Oban.insert/3` inside `Repo.transaction`).
- Ordering: a failing row holds back only the later rows of the same Job aimed at the same place (Tracker, Forge, or git). Whether open source can do this, or only Pro (chains, workflows, partitioned queues).
- Retries every `poll_interval`, giving up after 24 hours, retrying every given-up job at server start, and keeping finished jobs until their Job is deleted (the pruner).
- What Oban's own tables and processes add to startup and to a single-user server.
- Whether Oban's cron or scheduled jobs could drive the per-Job Ticket and Shipment checks every `poll_interval` (default 60 seconds), given cron's one-minute granularity.
- Licence of open source versus Pro.

## Answer

Researched against Oban v2.24.2 source and Oban Pro v1.8.3 docs. Open-source Oban meets some rules as they stand; ordering, the 24-hour give-up, the restart retry and the wait for a first try all need Taiichi code around it.

- **SQLite:** open source runs on SQLite with `Oban.Engines.Lite`, adding one table, `oban_jobs`. Oban sets no pragmas and uses Taiichi's own Repo, so WAL, the busy timeout and `synchronous = NORMAL` carry over. They are already `ecto_sqlite3`'s defaults. Oban's fetches and acks share SQLite's single writer with every save.
- **Same transaction:** yes, through `Ecto.Multi` or `Oban.insert` inside `Repo.transaction`. Oban notifies the queue before the commit, so the first try starts within the stager's one-second tick after the commit, not at once.
- **Ordering:** open source can't hold back later jobs. A queue with a limit of 1 runs the next job while a failed one waits to retry, and it orders every Job together. A hand-rolled gate fits: the worker snoozes while an earlier job of the same Job and place is pending, and gives itself up if that job was given up. Since v2.24 a snooze doesn't use an attempt. Pro's chains (`chain: [by: [args: [...]], on_discarded: :hold]`) match the rule exactly, but no first-party source says they run on SQLite. Pro's engine (partitioned queues) and Workflows don't run on SQLite, per Oban's core team on the Elixir Forum.
- **Retries:** a `backoff/1` returning `poll_interval` fits. Oban gives up by attempt count, so the 24 hours needs a time check in the worker. `Oban.retry_all_jobs/2` can retry given-up jobs at start, but it grants one more attempt only, so Taiichi must reset the 24-hour window itself, and must filter out completed jobs. The pruner is opt-in, so leaving it out keeps every job; deleting a Job deletes its jobs by an `args` query, with no foreign key. Every failure adds an entry to the job's `errors`: 1,440 a day at 60 seconds.
- **Startup and processes:** a migration that calls `Oban.Migration.up/1`. One supervisor tree: a notifier, a peer that is always leader, a stager that queries every second, and a producer per queue. Orphaned `executing` jobs are rescued only by the opt-in Lifeline, by age, so startup should reset them, then retry given-up jobs, before the queues start. Waiting for a first try has no open-source API; telemetry `[:oban, :job, :stop | :exception]` carries the result. Pro's Relay has `await`.
- **Cron:** five-field cron, minute resolution, crontab fixed at boot. It fits a `poll_interval` only when it is whole minutes that `*/n` can express. Pro's Cron changes entries at runtime but keeps that resolution. A self-rescheduling job (`schedule_in: poll_interval`), or a timer in each Job's existing process, works at any interval.
- **Licence:** open source is Apache 2.0. Pro is proprietary (Soren LLC): internal use only, no modification, no third-party access, and all copies deleted on termination, plus a two-year non-compete. It costs $150 a month or $1,620 a year per application, with no open-source discount.

Findings: `docs/research/oban-fit.md` on branch `research/oban-fit`, commit `fea32df`; local only, since the push failed without `ssh`.
