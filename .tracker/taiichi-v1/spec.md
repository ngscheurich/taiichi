# Taiichi v1

## About this spec

Taiichi is a "software factory" that keeps track of programming work. Work items come from a project tracker, such as the issues of a Forgejo repository. When a person asks, Taiichi takes one item and walks it through four fixed Stages: Plan, Build, Evaluate, Ship. At each Stage, someone writes a document saying what they did, and Taiichi saves it. At the end, the work goes up for code review as a pull request, and Taiichi watches the pull request until it is merged.

In v1, people do the work at every Stage. Taiichi keeps the record, enforces the order of the Stages, manages a Git branch for each piece of work, and talks to Forgejo. Later versions will let AI agents do Stages, so everything Taiichi stores must make sense to both people and programs.

This spec describes v1 well enough to split it into implementation tasks. It's written for a reader new to the project, whether a person or a coding agent. It decides nothing new: every rule comes from a decision ticket on the [Taiichi v1 map](map.md).

How to read it:

- **Capitalized words** such as Job, Stage and Attempt are domain terms with exact meanings. [Section 1](#1-words-used-in-this-spec) lists them, and [GLOSSARY.md](../../GLOSSARY.md) is the authority on them.
- **Sources** at the top of each section name the tickets the section comes from. Read them for the full reasoning and the options that were rejected.
- **Open** notes mark five questions still being decided in [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md). That ticket updates this spec when it closes.
- **[Readings made while assembling](#readings-made-while-assembling)**, at the end, lists the places where the tickets implied an answer without stating it.

## 1. Words used in this spec

### Domain terms

These come from [GLOSSARY.md](../../GLOSSARY.md).

| Term | Meaning |
|---|---|
| **Repo** | A Git repository that has opted in to Taiichi by running `taiichi init`. |
| **Tracker** | The system work items come from. In v1, always Forgejo issues. |
| **Ticket** | One work item in the Tracker, such as Forgejo issue #42. Taiichi reads Tickets and labels them, but doesn't own them. |
| **Job** | Taiichi's own record of one Ticket moving through the Stages: where it stands now and everything that happened. Each Job has a number, unique across the whole server and never reused. |
| **Pickup** | Starting a Job from a Ticket. Always a person's command; Taiichi never picks up Tickets on its own. |
| **Operator** | The person in the loop. In v1, the Operator picks up Tickets, writes every Stage Output, and decides what to do when Taiichi can't. |
| **Forge** | Where finished work goes for review. In v1, always Forgejo pull requests. |
| **Shipment** | The work under review on the Forge: one pull request per Job. Every Ship of a Job updates the same pull request. |
| **Report** | What the Forge says about the Shipment each time Taiichi checks: **Pending** (still under review, including approved but not merged), **Accepted** (merged), **Refused** (a reviewer requested changes), or **Closed** (closed without merging). |
| **Pipeline** | The fixed order every Job follows: Plan, Build, Evaluate, Ship, with loops back to Build, ending in Done or Abandoned. |
| **Stage** | One of the four steps where work happens: Plan, Build, Evaluate, Ship. |
| **Attempt** | One try at a Stage. Attempts are numbered within their Stage: Build 1, Build 2. A Stage repeats when its work is sent back, so a Job can have several Build Attempts. |
| **Position** | Where a Job stands: a Stage at an Attempt (such as "Build 2"), or one of the two ends. |
| **Done** | The end for a Job whose Shipment was Accepted. |
| **Abandoned** | The end for a Job the Operator dropped. |
| **Stage Input** | What a Stage receives when the Job enters it: earlier documents the person working the Stage should read. |
| **Stage Output** | The document a Stage produces. Submitting it ends the Attempt. Each Stage has its own kind: |
| ↳ **Blueprint** | Plan's Output: how the Ticket will be built. |
| ↳ **Build Notes** | Build's Output: what was built and how. |
| ↳ **Evaluation** | Evaluate's Output, carrying a **Verdict**: pass or fail. |
| ↳ **Manifest** | Ship's Output: the pull request's description, plus what was shipped and where. |
| **Ticket Snapshot** | A copy of the Ticket taken once, at Pickup. It never changes, so the Job is always held to what the Ticket said when work started. |
| **Feedback** | The comments reviewers left on the pull request, gathered when it is Refused and handed to the next Build. |
| **Foreign commit** | A commit someone other than Taiichi put on the Job's branch on Forgejo, such as a fix a reviewer pushed. It counts as part of the Job's work. |

### Other terms

These aren't domain terms, but the spec uses them throughout.

| Term | Meaning | Explained in |
|---|---|---|
| **worktree** | A second checkout of a Git repository in its own folder (`git worktree add`). Each Job gets one, so Jobs don't disturb each other or the main checkout. | [Section 5](#5-the-pipeline) |
| **Git common dir** | The `.git` folder of the main clone, which all its worktrees share. Taiichi uses it to recognize a Repo from any of its worktrees. | [Section 12](#12-configuration-and-taiichi-init) |
| **save** | The single database transaction that records a change to a Job. | [Section 7](#7-doing-each-action-safely) |
| **outbox** | A table of things Taiichi still has to do in Git or on Forgejo after a save, retried until they succeed. | [Section 9](#9-the-outbox) |
| **checker** | A process per active Job that looks at its Ticket and pull request on Forgejo every minute. | [Section 8](#8-watching-for-changes-the-checker) |
| **flag** | A warning the CLI shows on a Job, such as "Ticket closed". Flags never move a Job. | [Section 8](#8-watching-for-changes-the-checker) |
| **ready label**, **position label** | Forgejo labels Taiichi uses: a ready label marks an issue as available for Pickup, and a position label shows which Stage its Job is at. | [Section 10](#10-forgejo-as-the-tracker) |
| **Ship marker**, **feedback cursor** | Two saved Forgejo IDs that tell Taiichi which reviews and comments are new. | [Section 7](#submit-at-ship), [section 11](#11-forgejo-as-the-forge) |
| **base commit**, **judged commit** | The default-branch commit a Job's branch starts from, and the commit an Evaluation looked at. | [Section 6](#6-stage-documents) |

## 2. One Job from start to finish

This walk-through shows the whole system once, without the edge cases. The rest of the spec fills them in.

The Repo is `ngs/widgets` on a Forgejo instance. Issue #42, "Add a dark mode toggle", carries the label `taiichi/ready`.

1. **Pickup.** The Operator runs `taiichi tickets` and sees #42 among the issues ready to start. They run `taiichi pickup 42`. Taiichi creates Job 7, saves a Ticket Snapshot of #42, and creates the branch `taiichi/42-j7-add-a-dark-mode-toggle` with a worktree for it. On Forgejo it swaps the issue's `taiichi/ready` label for `taiichi/plan` and comments that Job 7 started. Job 7 is now at Plan.
2. **Plan.** The Operator runs `taiichi input` to read Plan's Input, which is the Ticket Snapshot. They run `taiichi submit`, write the Blueprint in their editor, and save. Job 7 moves to Build 1, and the issue's label becomes `taiichi/build`.
3. **Build.** The Operator changes code in the Job's worktree and commits. They submit Build Notes. Taiichi refuses if the worktree has uncommitted changes, and records which commits this Attempt made. Job 7 moves to Evaluate.
4. **Evaluate.** The Operator checks the work and submits an Evaluation with `--fail`. Job 7 goes back to Build 2, whose Input includes the failing Evaluation. After more work, a second Evaluation passes with `--pass`, and Job 7 moves to Ship 1.
5. **Ship.** The Operator writes the pull request's description and submits it as the Manifest. Taiichi pushes the branch to Forgejo, opens a pull request, and waits for review.
6. **Review.** Every minute, Taiichi's checker looks at the pull request. A reviewer requests changes, so the Report is Refused. Job 7 goes back to Build 3, whose Input includes the reviewer's comments as Feedback. The Operator builds and evaluates again. Ship 2 updates the same pull request.
7. **Done.** The reviewer merges the pull request, so the Report is Accepted. Job 7 is Done. Taiichi labels the issue `taiichi/done`, comments, closes it, and removes the worktree. The branch and every document of Job 7 are kept.

At any point, `taiichi abandon` ends a Job as Abandoned. Its issue gets its ready label back, so it can be picked up again.

## 3. What v1 covers

Sources: [Chart the v1 scope](issues/01-chart-the-v1-scope.md), [The executor seam](issues/19-executor-seam.md).

v1 covers:

- Opting a Git repository in with `taiichi init`.
- Listing the Tickets ready for Pickup, and Pickup itself.
- Enforcing the Pipeline, including both loops back to Build.
- Saving every Stage Input and Stage Output. Every Attempt is kept; nothing is overwritten.
- A branch and a worktree for each Job.
- Updating the Ticket's labels and comments, shipping the work as a pull request, and watching both the Ticket and the pull request for changes.
- Forgejo as both Tracker and Forge, on an instance the user hosts.

v1 builds nothing for future AI executors unless adding it later would be costly, meaning it would break stored data or change an existing API method. Everything else waits; see [section 17](#17-not-in-v1).

Taiichi runs on one machine for one user. Remote deployment is out of scope.

## 4. How the pieces fit

Sources: [Chart the v1 scope](issues/01-chart-the-v1-scope.md), [HTTP over a Unix socket](issues/04-http-over-unix-socket.md), [Revisit the transport and protocol](issues/14-revisit-transport-and-protocol.md), [CLI language](issues/11-cli-language.md).

Taiichi has two programs:

- **The server**, written in Elixir. It runs for a long time and serves every Repo on the machine. The Operator starts it by hand in a terminal. It holds all the rules and is the only program that writes Job data.
- **The CLI**, `taiichi`, written in Go. It holds no rules of its own: it works out which Repo and Job the user means, sends a request to the server, and prints the answer for people. It also opens the user's editor for writing Stage Outputs, and always sends the text itself, never a file path. When it can't reach the server, it says so plainly.

**How they talk:** HTTP with JSON bodies, over a Unix domain socket. A Unix domain socket is a special file that programs connect to the way they'd connect to a network port. The socket sits in a folder only the user can open (mode `0700`), so no other account on the machine can connect, and web pages in a browser can't reach it at all. That matters because the server holds Forgejo access tokens and creates branches. TCP on localhost was rejected because it would need a token or `Origin` checks to get the same protection. JSON-RPC was rejected because Bandit and Plug, the Elixir web libraries, already handle routing, parsing and errors for HTTP.

**Repo layout:** the server's Mix app lives in `server/` and the CLI in `cli/`, side by side. The root holds what both share: `.tracker/`, `docs/`, `GLOSSARY.md`, and `mise.toml` with both toolchains. Keeping them in one repository lets one commit change an API method and the CLI code that calls it.

## 5. The Pipeline

Sources: [Chart the v1 scope](issues/01-chart-the-v1-scope.md), [Stage Inputs and Stage Outputs](issues/05-stage-inputs-and-outputs.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md).

Every Job follows the same fixed Pipeline. Nothing about it is configurable.

| From | What happens | To |
|---|---|---|
| (no Job yet) | the Operator picks up the Ticket | Plan 1 |
| Plan | the Operator submits the Blueprint | Build 1 |
| Build | the Operator submits Build Notes | Evaluate |
| Evaluate | the Operator submits an Evaluation with a passing Verdict | Ship |
| Evaluate | the Operator submits an Evaluation with a failing Verdict | the next Build Attempt |
| Ship | the Operator submits the Manifest | Ship, now waiting on the Shipment |
| Ship, waiting on the Shipment | the checker sees the Shipment Refused | the next Build Attempt, with Feedback |
| any Stage, once there is a Shipment | Taiichi sees the Shipment Accepted | Done |
| any Stage | the Operator abandons the Job | Abandoned |

Rules that follow from it:

- **Submitting moves the Job at once.** A valid Stage Output ends the Attempt, and the Job enters the next Stage in the same save.
- **Only Build repeats in v1.** It repeats when an Evaluation fails or a reviewer refuses the Shipment. A Job never returns to Plan; a wrong Blueprint means abandoning the Job and picking the Ticket up again.
- **Done comes only from the Shipment being Accepted,** and Taiichi can notice that at any Stage. For example, a reviewer may merge the pull request while the Job is back in Build. Taiichi notices through the checker ([section 8](#8-watching-for-changes-the-checker)), or when the Operator abandons the Job or submits at Ship ([section 7](#7-doing-each-action-safely)).
- **A Closed Shipment never moves the Job.** A closed pull request might be a mistake or might mean "start over", so a person decides. Away from Ship, the Job carries on with a flag. At Ship, Taiichi refuses to ship until the Operator reopens the pull request on Forgejo; a Job already waiting on a Closed Shipment stays waiting. The Operator can abandon the Job instead. Taiichi never abandons a Job on its own.
- **One active Job per Ticket,** across the whole server. Many Jobs can be active at once.
- **Ends keep history.** At Done or Abandoned, Taiichi removes the worktree but keeps the branch and every document.
- **Plan prints plainly.** A Stage with only one Attempt prints without its number in the CLI ("Plan", not "Plan 1"). The data and the API always number it.

## 6. Stage documents

Sources: [Stage Inputs and Stage Outputs](issues/05-stage-inputs-and-outputs.md), [Server internals](issues/12-server-internals.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [Forgejo Forge](issues/08-forgejo-forge.md).

### Documents

Taiichi stores everything about a Job's work as **documents**. Each document has three parts:

- `kind`: which sort of document it is (`ticket_snapshot`, `blueprint`, `build_notes`, `evaluation`, `manifest`, or `feedback`).
- `body`: Markdown text written by a person, or copied from Forgejo.
- `meta`: structured values, such as an Evaluation's Verdict. The server fills these in; people never write them by hand.

Documents are **written once** and never changed. Anything that changes later, such as the latest Report on a pull request, is stored outside them.

When the Operator submits a Stage Output, they send the Markdown body plus at most one extra value: the Verdict at Evaluate, or an optional pull request title at Ship. The server works out everything else in `meta`, including the Job, Stage, Attempt and submission time. The server only asks for values it acts on.

| Kind | Body | `meta` (filled in by the server) | Checked when submitted |
|---|---|---|---|
| Ticket Snapshot | the Ticket's body, word for word, without comments | Ticket ID, URL, title, labels, when the copy was taken | (written by the server at Pickup) |
| Blueprint | how the Ticket will be built | (identity only) | the body isn't empty |
| Build Notes | what this Attempt built and how | this Attempt's commits, any change of base, and any foreign commits (see below) | the body isn't empty; the worktree has no uncommitted changes (`409 worktree_dirty`) |
| Evaluation | the judgment | the Verdict (`pass` or `fail`) and the judged commit | the body isn't empty; a Verdict is given; the worktree's head hasn't moved since the Build Notes being judged (`409 head_moved`) |
| Manifest | the pull request's description | the title, the pull request's URL and number, target branch, the shipped and judged commits, and the feedback cursor | the body isn't empty; a title, if given, isn't empty |
| Feedback | one section per comment: author, file and line if inline, whether it's resolved, text | the pull request, the commit reviewed, the Ship Attempt it came from, and the feedback cursor it moved to | (written by the server when a Shipment is Refused) |

- A Stage Output is one document. v1 has no attachments.
- The Manifest's title defaults to the Ticket Snapshot's title. Ship 2 and later may leave it out to keep the current title.
- The **judged commit** is the worktree's head when the Evaluation was submitted. Ship checks against it ([section 7](#submit-at-ship)).

### Which commits belong to a Build Attempt

Build Notes record the commits their Attempt made, so a reader can see what each Build changed. The rules make sure every commit on the branch belongs to exactly one Build Attempt.

- **Where the range starts:** at the last commit Taiichi pinned for the Job. That's the **base commit** for Build 1: the head of the default branch at Pickup, where the branch was created. After a failing Evaluation, it's that Evaluation's judged commit. After a Refusal, it's the commit Ship pushed.
- **Where it ends:** the worktree's head when the Build Notes are submitted.
- **Commits made between Attempts,** for example while the Job waited for review, fall into the next Build's range.
- **Whose commits count:** every commit that came into the branch during the Attempt, whoever wrote it, except commits from the default branch. If the Operator rebases onto a newer `main`, the commits that came from `main` are left out. Build Notes record the move instead, such as `base: main@1111 → main@2222`. "From the default branch" means reachable from the local `origin/<default>` ref.
- **Foreign commits** in the range are listed with their sha and author. A commit is foreign if it's reachable from the local `origin/<job branch>` ref but not from any commit Taiichi pushed. `git pull` and Taiichi's own pushes keep that ref current, so this needs no Forgejo call.
- **An empty range is fine.** Sometimes nothing needs to change: an Evaluation failed because its test environment was stale, or Feedback needed an answer rather than a fix. The Attempt still writes fresh notes saying why.
- **The range only labels commits.** Evaluate judges the whole project at the head, not just the range.
- **Old commit IDs may stop resolving.** If the Job's history is rewritten, Git eventually deletes the old commits. Taiichi only stores those IDs for the record and never acts on them.

### What each Stage receives

A Stage Input is an ordered list of documents. Taiichi stores the list when the Attempt starts and never recomputes it, so a later change to these rules never changes what an old Attempt received. Documents are referenced, not copied.

| Stage | Input, in order | Why |
|---|---|---|
| Plan | the Ticket Snapshot | Plan works out how to build what the Ticket asks for. |
| Build | the Blueprint, then why the Job came back, if it did: the failing Evaluation or the Feedback | The worktree already holds the earlier work, so Build needs only the plan and the latest reason. |
| Evaluate | the Ticket Snapshot, the Blueprint, then every Build Notes in order, each followed by the failing Evaluation or Feedback that sent the Job back | Each Build Notes covers only its own Attempt, so the evaluator needs the whole story. |
| Ship | the same as Evaluate, ending with the passing Evaluation | The Manifest describes the change as a whole. |

> **Open:** the API reads a stored Stage Output by its Stage and Attempt, but has no way to read the Ticket Snapshot or an older Attempt's Feedback once the Job has moved on. Reading the current Stage Input doesn't help either: it's refused on a Job that has ended. See [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md).

## 7. Doing each action safely

Sources: [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [Server internals](issues/12-server-internals.md), [`taiichi init` and configuration](issues/09-init-and-configuration.md).

### The problem

Most actions touch three places: Taiichi's database, Git, and Forgejo. A database transaction can't include Git or Forgejo. Forgejo can be unreachable, and the server can stop between any two steps. Each action has to leave things in a state a retry can finish, without duplicating work: no second pull request, no lost review.

### The rule

Each action that moves a Job has exactly one **save**: one database transaction. Its other steps fall on either side of the save.

- **Before the save:** steps whose results the save needs. Each must be safe to repeat. If one fails, nothing is saved, the command fails with the error, and the Operator runs the same command again.
- **After the save:** every other step, written as rows in the [outbox](#9-the-outbox) inside the save itself. The command succeeds once the save lands, and the outbox finishes the rest, retrying until it succeeds.
- **No transaction stays open** while Taiichi waits on Git or Forgejo.
- **A retry checks the world, not a progress record.** It asks Git or Forgejo whether a step already happened. That way, a step that succeeded just before the server stopped is still found.

| Action | Before the save | The save records | After the save (outbox) |
|---|---|---|---|
| Pickup | read `taiichi.toml`; fetch the Ticket; check it's eligible, the labels exist, and the branch name is free; read the base commit | the Job, the Ticket Snapshot, Plan 1 and its Input | create the branch and worktree; set labels; comment on the issue |
| Submit at Plan, Build, Evaluate | the Stage's checks ([section 6](#documents)) | the Stage Output, and the next Attempt with its Input | set labels |
| Submit at Ship | read the pull request; check the judged commit and the branch; save the Ship marker if pushing; push; find or open the pull request | the Manifest | set labels; on Ship 2 and later, update the description, comment, and re-request review |
| Shipment Refused | read the reviews and comments | the Feedback, and the next Build Attempt with its Input | set labels |
| Shipment Accepted | read the pull request | the Job ends as Done | set labels; comment; close the issue; remove the worktree |
| Abandon | read the pull request if there is one; refuse uncommitted changes | the Job ends as Abandoned | close the pull request; restore ready labels; comment; remove the worktree |

**What a Forgejo error means:** `502 forgejo_failed` always means nothing was saved. Only Ship can leave work done on Forgejo with nothing saved (a push, a new pull request), and running `submit` again finishes it. Outbox steps never cause a `502`; their failures show as a warning on the Job instead.

### Pickup

1. **Read the config.** Read `taiichi.toml` fresh from the clone's working tree. If it names a different Forgejo repository than the one the Repo registered, refuse with `409 forge_location_changed` and ask the Operator to run `taiichi init`.
2. **Check the Ticket.** Fetch it and check it's eligible ([section 10](#10-forgejo-as-the-tracker)). If not, refuse and say why: `409 ticket_not_eligible`, or `409 ticket_has_active_job`.
3. **Check the branch name.** Make sure the Job's branch name isn't already taken, in the clone or on Forgejo (`409 branch_taken`), so Pickup doesn't fail halfway.
4. **Read the base commit:** the head of the local `origin/<default>`.
5. **Save,** in one transaction started with `BEGIN IMMEDIATE`: the Job, with a copy of the Repo's settings ([section 12](#12-configuration-and-taiichi-init)), the Ticket Snapshot, the Job's `tickets` row ([section 13](#13-storage)), Plan 1 with its Input, and the outbox rows.
6. **Start** the Job's process and checker, wait for the outbox's first try, and reply.

If two Pickups of the same Ticket race, the database's unique index on active Jobs lets only one save land. The other gets `409 ticket_has_active_job`.

> **Open:** the branch name contains the Job number, but the database assigns the number only in step 5, after step 3 has checked the name. See [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md).

### Submit at Plan, Build and Evaluate

The server runs the checks for the Stage's document ([section 6](#documents)), then saves the Output and starts the next Attempt. The submission names the Stage and Attempt it was written for. If the Job has moved on since, for example because the same Output was already submitted from another terminal, the server refuses with `409 job_moved`. That way, Build Notes written for Build 2 can never land on Build 3.

### Submit at Ship

**Ship 1** is the Job's first Ship Attempt; it opens the pull request. A Job only comes back to Ship after a Refusal sent it to Build, so **Ship 2 and later** always update that same pull request.

1. **Read the pull request** (Ship 2 and later), by the number saved for the Shipment, before anything else:
   - Merged: the Job moves to Done, and the submission is refused with `409 shipment_accepted`.
   - Closed: refused with `409 shipment_closed` until the Operator reopens it on Forgejo. Forgejo closes a pull request whose branch was deleted, and its web page won't reopen one while the branch is missing. So if the branch is gone, Taiichi first pushes it back from the worktree, then refuses with "Branch restored. Reopen #42 on Forgejo, then submit again."
   - Open: carry on.
2. **Check the judged commit.** If the worktree's head isn't the commit the passing Evaluation judged, refuse with `409 head_moved`. This stops unevaluated work from being shipped by accident. The Operator can override it with `--unjudged`, which exists for foreign commits found only at Ship: before a Job ships, nothing sends it back to Build to have them evaluated. Only a Refused Shipment returns a Job from Ship to Build, and that needs a Ship first.
3. **Check the branch on Forgejo** and choose how to push. Git's `--force-with-lease` overwrites the branch on Forgejo only if it's still at the commit Taiichi expects. If someone else pushed in between, Git refuses rather than erase their work.

   | Forgejo's branch is | Push expecting | Result |
   |---|---|---|
   | at the commit Taiichi last shipped | that commit | replaces it. This is the only case where the Job's own rewritten history (a rebase or squash) can overwrite Forgejo's. |
   | contained in the worktree's head | Forgejo's current head | a normal fast-forward. A push that lands between the check and the push is still refused. |
   | missing | no branch | recreates it |
   | anything else | (no push) | `409 branch_moved`: someone pushed commits the worktree lacks. Nothing is saved; the Operator pulls them in first. |

4. **Save the Ship marker** if a push is needed. A "request changes" review on Forgejo stays in force after later pushes. Without a marker, an old refusal would send every new Ship straight back to Build. So each time Taiichi ships, it notes the ID of the newest review on the pull request. Forgejo's review IDs only go up, so only a review with a higher ID can make the Shipment Refused. That noted ID is the **Ship marker**, kept in `shipments.ship_marker`.
   - **Ship 2 and later** save the marker *before* pushing. If Taiichi pushed first and the server stopped before the save, a reviewer could request changes on the new commits in the gap. The retry would then take that review as the newest, make it the marker, and never count it.
   - **On a retry,** if the push hasn't landed yet, Taiichi reads and saves the marker again, since any review in the meantime was about the old commits. If the push has landed, it keeps the saved marker.
   - **Ship 1** saves no marker before pushing. The Job has no `shipments` row yet, and a brand-new pull request has no reviews, so every review counts. The row is created with an empty marker in step 7.
5. **Push** over HTTPS as Taiichi's Forgejo account, with `--force-with-lease=refs/heads/<branch>:<sha>`. It always names the expected commit (empty when Forgejo has no branch yet), never the bare flag. If Forgejo already has the local head, Git reports "Everything up-to-date" and succeeds, so a retry after a push that landed needs no special handling.
6. **Find or open the pull request** (Ship 1 only). A retry must not open a second one, so Taiichi first looks for an existing one with `GET /pulls/{base}/{head}`. It adopts it only if it's open and was opened by Taiichi's account. A closed or merged one is refused with `409 shipment_closed`; the branch name contains the Job number, so only this Job could have used it. If there's none, Taiichi opens one ([section 11](#11-forgejo-as-the-forge)).
7. **Save** the Manifest and the outbox rows. On Ship 1, also create the Job's `shipments` row: the pull request's URL and number, with an empty marker.

### Abandon

- **Allowed** from any Position while the Job is active. Unlike `submit`, it doesn't check which Attempt the Operator meant, since its confirmation prompt is only seconds old.
- **Uncommitted changes** in the worktree are refused with `409 worktree_dirty`, so work isn't lost by accident. `taiichi abandon --discard` overrides this, and the worktree is then removed with `--force`. The confirmation prompt mentions the uncommitted changes.
- **A merged pull request wins.** If the Job has a pull request, Abandon reads it first. If it's merged, the Job moves to Done instead and Abandon is refused with `409 shipment_accepted`. If Forgejo can't be reached, Abandon fails with `502`, because it can't tell. If the pull request is open, the outbox closes it with a comment saying the Job was abandoned and the branch is kept. That way nobody merges work with no Job behind it. A Job that never shipped doesn't call Forgejo at all.
- **The issue stays open** and gets its ready labels back ([section 10](#10-forgejo-as-the-tracker)), so it can be picked up again.

### Done

- The same steps run whichever Stage the Job was in: labels, a comment, closing the issue, removing the worktree.
- **A worktree with uncommitted changes is kept.** Taiichi never deletes uncommitted work unless the Operator says to. The removal step ends `skipped`, the Job's `worktree` stays set, and `show` says "worktree kept: uncommitted changes".
- **An unfinished Attempt** has no Stage Output. `show` prints it as, for example, "Build 2 · unfinished". If the branch has commits beyond what was last shipped, `show` adds "2 unshipped commits on `taiichi/42-j7-…`".

> **Open:** an ended Job takes no commands, so the Operator has no way to say a kept worktree can go. See [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md).

## 8. Watching for changes: the checker

Sources: [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [Forgejo Forge](issues/08-forgejo-forge.md), [Oban or the hand-rolled outbox](issues/21-oban-or-hand-rolled-outbox.md), [Watching a Job live](issues/18-watching-a-job-live.md).

People change things on Forgejo while a Job runs: they edit the issue, comment, review, merge or close the pull request, or push to the branch. Taiichi finds out by looking. Forgejo's webhooks would push changes to it, but they aren't used.

### How it works

Each active Job has a **checker**, a process that runs beside the Job's own process. Every `poll_interval` (60 seconds by default), it reads the Job's Ticket and, once there is one, its pull request. It hands what it finds to the Job's process, which saves it and makes any move. The checker never writes to the database itself; the Job's process stays the only writer. Checkers start with their Job's process, including when the server starts, and check at once.

The checker is the only thing that changes a Job without a command. Its timer is a plain `Process.send_after/3`; Oban's scheduled jobs were weighed and rejected.

Each check costs two Forgejo calls per Ticket and three per pull request, even when nothing changed. Reading each one directly keeps the code simple, so the earlier plan of one repository-wide query was dropped.

### What it looks at on the Ticket

It calls `GET /issues/{n}`, and `GET /issues/{n}/comments?since=<seen point>`.

- **Closed or deleted:** the Job is flagged "Ticket closed" or "Ticket deleted" (a deleted issue answers 404). The Job carries on, and the Operator decides whether to abandon it. "Ticket closed" clears if the issue is reopened.
- **Edited:** the title, the body, and any labels Taiichi doesn't manage are compared with the Ticket Snapshot. Stage Inputs never change, because the Snapshot is what the Job is held to. Comparing the current state, rather than watching for edit events, means an edit that is undone clears itself.
- **New comments** from anyone but Taiichi's account are counted.
- **Taiichi's own changes never raise a flag.** Its labels are left out of the comparison, its comments come from its own account, and it closes the issue only at Done, after checking has stopped.

### What it looks at on the pull request

It reads the pull request, its reviews, and its comments after the feedback cursor. What counts as each Report is defined in [section 11](#report).

- **Accepted,** at any Stage: the Job moves to Done.
- **Refused,** only while the Job is waiting at Ship: the Job moves to the next Build Attempt, with Feedback. At other times, reviews are ignored. They're older than the next Ship marker, and their comments reach the next Feedback anyway.
- **Closed or reopened:** the Job's latest Report changes, and with it the "Shipment closed" flag. Nothing else happens ([section 5](#5-the-pipeline)).
- **The branch moved:** each check saves the pull request's head commit as `branch_head`. If the worktree's head doesn't contain it, someone pushed commits the worktree lacks, and the Job is flagged: "The branch has commits your worktree lacks: run `git pull --rebase` in the worktree." Taiichi's own pushes never raise this flag. Build and Evaluate can still be submitted, so an evaluator can fail the Verdict to take the new commits back to Build. Only Ship refuses ([section 7](#submit-at-ship)).
- **New comments** while the Job is away from Ship are counted, from anyone but Taiichi's account. They don't change any Stage Input; they still reach the next Feedback if the Shipment is Refused, and are never handed over twice.

### When a check fails

If Forgejo can't be reached, nothing changes: flags keep their last values, and the Job records when checks started failing. The CLI shows it, for example: `Shipment pending · reported 11:58 · reports failing since 12:03`.

### Marking things seen: `ack`

Without a way to say "I've seen this", an edited Ticket would stay flagged forever, and a second edit would look exactly like the first. `taiichi ack` marks what's new on a Job as seen: the Ticket's edits and comments, and the pull request's new comments. The last check's state becomes the **seen point**, and later checks compare against it.

- `ack` never calls Forgejo; it records what the last check saw.
- It doesn't affect the feedback cursor, so the next Refusal still gathers every comment since the last Feedback.
- Flags about a state, rather than news, can't be acked: "Ticket closed", "Ticket deleted", "Shipment closed", and the branch-head flag. They clear when their cause goes away.

## 9. The outbox

Sources: [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [Oban or the hand-rolled outbox](issues/21-oban-or-hand-rolled-outbox.md).

The **outbox** is a table of steps Taiichi still has to take in Git or on Forgejo after a save. Each save writes its rows in the same transaction, so a move is never recorded without its follow-up work, even if the server stops right after.

The outbox is written by hand, not built on the Oban job library. [Oban or the hand-rolled outbox](issues/21-oban-or-hand-rolled-outbox.md) explains why: open-source Oban met three of the outbox's eight rules and would have slowed replies by up to a second, and Oban Pro isn't known to run on SQLite.

- **Rows are instructions, not history.** A row says what to do, such as "add these labels to issue #42". The Job's Attempts and documents already record what happened. Each row's details are worked out at save time, from the Job's copy of its settings.
- **The kinds of steps in v1:**
  - on the Ticket: set labels, comment, close the issue;
  - on the pull request: update its description (and title), comment, re-request review, close it;
  - in Git: add a worktree (creating the branch at the base commit), remove a worktree.
- **One sender** sends the rows. It runs apart from the Job processes, because ended Jobs have no process and Done still has work after its save. After each save, the Job's process tells the sender to start.
- **Order:** a failing row holds back only the later rows of the same Job aimed at the same place (the Tracker, the Forge, or Git). Other Jobs, and the Job's other places, carry on.
- **Retries:** every `poll_interval`. After 24 hours of failing, a row is given up, along with the rows waiting behind it. Every server start retries all given-up rows, so fixing a token or bringing Forgejo back takes effect on restart, with no special command.
- **Finished rows are kept** until their Job is deleted, as a record of what Taiichi did outside.
- **No duplicate comments:** Taiichi skips a comment if its account already posted the same text on that issue or pull request. A retry after a comment that landed doesn't post it twice.
- **A deleted issue:** when a Ticket step gets a 404, the sender checks whether the repository still answers. If it does, the issue was deleted, and the row ends `skipped`. If not, the row keeps failing as usual, since Forgejo also answers 404 when a token loses access. Done and Abandon still complete.
- **Replies wait briefly.** `pickup`, `submit` and `abandon` wait for the first try of their new rows, then succeed either way. If anything is still pending, the Job shows it, the CLI prints a warning under its normal output, and `jobs` marks the Job.

## 10. Forgejo as the Tracker

Sources: [Forgejo Tracker rules](issues/07-forgejo-tracker-rules.md), [`taiichi init` and configuration](issues/09-init-and-configuration.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [Forgejo API for the Tracker and Forge](issues/02-forgejo-api.md).

### The Tracker interface

Taiichi's core talks to the Tracker through an interface (an Elixir behaviour), so other Trackers can be added later. The interface speaks in Positions, never labels:

- **`moved(old_job, new_job)`** is called after every save that moves a Job, with `nil` as the old Job at Pickup. It compares the two Positions and writes outbox rows to update the Ticket.
- **`check`** returns the Ticket's state (`open`, `closed`, or `deleted`), what differs from the Snapshot and from the seen point (title, body, labels), and any new comments. Each Tracker leaves its own markers, such as Taiichi's labels, out of the comparison.

The core treats Ticket IDs as opaque strings and keeps only the rules that hold for every Tracker. Everything below is how the Forgejo Tracker works.

### Ticket IDs and branch names

- **A Ticket ID** is the bare issue number, such as `42`. Anything else is rejected, including `#42`, because the shell treats a word starting with `#` as a comment.
- **A branch name** is `taiichi/<ticket>-j<job>-<slug>`, such as `taiichi/42-j7-add-login-page`. The Job number keeps names unique, because branches outlive their Jobs and a Ticket can be picked up again. The slug comes from the Ticket Snapshot's title: lowercased, with anything that isn't a letter or digit turned into a hyphen, and cut to a fixed length.

### Which Tickets can be picked up

An issue is **eligible** when it's open, isn't a pull request, carries at least one of the Repo's ready labels, and has no active Job. Forgejo's label filter returns only issues carrying *all* the labels given, so Taiichi queries once per ready label and merges the results.

### Labels

Forgejo labels show where each Ticket stands, right in Forgejo's own issue list.

- **Ready labels** (default `taiichi/ready`) mark issues waiting to be picked up. Pickup removes every ready label from the issue. Abandon puts back the ones the Ticket Snapshot recorded, so the issue is eligible again.
- **Position labels** show the Job's Stage: by default `taiichi/plan`, `taiichi/build`, `taiichi/evaluate`, `taiichi/ship` and `taiichi/done`. Each can be renamed or turned off. There's no Abandoned label: the issue gets its ready labels back, and an issue showing both "ready" and "abandoned" was found confusing. The Abandoned comment and the Job's history record it.
- **Each move sets Taiichi's labels exactly.** It removes every ready label and every other position label, then adds the new position label. Abandon removes every position label and adds back the ready labels.
- **Taiichi touches only its own labels.** It adds them one at a time (`POST …/labels`) and removes them one at a time (`DELETE …/labels/{id}`), and never replaces the whole set with `PUT`. Labels people added are left alone.
- **Labels are never commands.** Adding a position label by hand doesn't move the Job, and re-adding a ready label to an issue with an active Job doesn't allow a second Pickup. A person's relabelling lasts until the next move.
- **Labels must exist.** Forgejo silently ignores unknown label names, so a typo would quietly do nothing. The server checks that every configured label exists and fails clearly if one doesn't. It never creates labels; only `taiichi init` offers to.
- **Exclusive labels:** Forgejo can mark labels as exclusive within a scope (the part of the name before the last `/`). Adding one removes the issue's other exclusive labels in that scope. `taiichi init` creates Taiichi's labels as exclusive labels in the `taiichi` scope, so an issue shows one at a time.

### Comments, assignees, closing

- **Comments** on Pickup (the Job number and branch), on Done (the merged pull request's URL), and on Abandoned (the branch is kept). Each can be turned off per Repo. There are no comments for the moves in between, because the position label shows them.
- **No assignee changes.** Taiichi doesn't know which person is working a Stage, and Forgejo can only replace the whole set of assignees at once.
- **Closing:** at Done, the Tracker closes the issue itself. The pull request carries no closing keyword such as `Closes #42`, so the Tracker and Forge code stay independent.

## 11. Forgejo as the Forge

Sources: [Forgejo Forge](issues/08-forgejo-forge.md), [Does the Forge interface fit GitHub and GitLab?](issues/03-forge-interface-fit.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [Forgejo API for the Tracker and Forge](issues/02-forgejo-api.md).

### The Forge interface

The Forge is also reached through an interface with two operations, checked against GitHub's and GitLab's APIs as well as Forgejo's:

- **Ship** puts a Job's branch up for review, or updates it, and produces what the Manifest records.
- **Report** says whether the Shipment is Pending, Accepted, Refused (with Feedback), or Closed, counting only reviews since the latest Ship.

Everything below is how the Forgejo Forge works. How Ship pushes and avoids duplicates is in [section 7](#submit-at-ship).

### The pull request

- **Ship 1** opens a pull request against the repository's default branch, which it reads from Forgejo. The title is the one submitted, or the Ticket Snapshot's. The body is the Manifest's body word for word, followed by a line `Ticket: <url>` linking the issue. There's no closing keyword, draft marker, label, assignee or requested reviewer.
- **Ship 2 and later** replace the description with the new Manifest's body, since it should describe the change as it now stands. The title changes only if one is given. Taiichi also comments to mark the new round, such as "Shipped again: `abc123..def456`", and asks everyone who refused last time to review again. All three are outbox rows.

### Report

- **Accepted:** the pull request is merged (`merged: true`).
- **Refused:** someone's most recent review is newer than the Ship marker, asks for changes (`state: "REQUEST_CHANGES"`), and hasn't been dismissed. Which commit the review names doesn't matter: Forgejo records whatever commit the reviewer's page was showing, which can be out of date.
- **Closed:** closed without merging.
- **Pending:** anything else, including comment-only reviews and approvals that haven't been merged.

### Feedback and the feedback cursor

When a Shipment is Refused, Taiichi gathers **Feedback**: everything new on the pull request from anyone but Taiichi's account. That covers review bodies (including comment-only reviews and approvals with text), inline comments with whether they're resolved, and ordinary comments.

"New" means after the **feedback cursor**, the ID of the last item carried into a Feedback. Each Manifest records the cursor when it shipped, and each Feedback records where it moved the cursor to. The cursor is separate from the Ship marker on purpose. A second reviewer's comments posted while the Job was back in Build still reach the next Feedback, even though they're older than the next Ship marker. Forgejo's IDs only go up, so Taiichi never compares clocks.

### The bot account

Forgejo doesn't let a pull request's author approve it or request changes on it. Taiichi opens every pull request with its token, so the token must belong to a dedicated bot account that never reviews. If it belonged to a reviewer, that reviewer's refusals would be blocked. Taiichi can't detect this, so it's documented, and the Operator can see which account Taiichi acts as.

> **Open:** [Forgejo Forge](issues/08-forgejo-forge.md) says `taiichi status` shows that account, but the `status` API method doesn't return it; only `init` does. See [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md).

### Forgejo version and token

The Forgejo behaviour above was checked against Forgejo v17.0.0 and the v15 long-term-support release, with pull request behaviour also read at v13.0.5. The user's instance version isn't confirmed yet; `GET /api/v1/version` tells. A personal access token limited to the repository, with `read:issue`, `write:issue`, `read:repository` and `write:repository`, covers everything v1 does.

## 12. Configuration and `taiichi init`

Sources: [`taiichi init` and configuration](issues/09-init-and-configuration.md), [HTTP over a Unix socket](issues/04-http-over-unix-socket.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [HTTP API shape](issues/10-http-api-shape.md).

### The Repo's config: `taiichi.toml`

Each Repo has a `taiichi.toml` at its root, committed so the whole team shares it. It says where the Repo lives on Forgejo and how Taiichi should label and comment. Only the server reads it; the CLI never parses it. Unknown keys are an error, so a typo fails loudly.

```toml
[forge]
type = "forgejo"
url  = "https://forgejo.example"
repo = "ngs/widgets"

[tracker]
type = "forgejo"
# url and repo default to the Forge's
ready_labels = ["taiichi/ready"]          # the default

[tracker.position_labels]   # each defaults to taiichi/<position>; false turns it off
plan     = "taiichi/plan"
build    = "taiichi/build"
evaluate = "taiichi/evaluate"
ship     = "taiichi/ship"
done     = "taiichi/done"

[tracker.comments]          # each defaults to true
pickup    = true
done      = true
abandoned = true
```

- `type` picks the Tracker or Forge code. Only `forgejo` exists in v1.
- **Each Job keeps a copy** of these settings, taken at Pickup, and runs under that copy. Background work, such as checking and the outbox, must not depend on whatever happens to be checked out in the clone. To change a running Job's settings, abandon it and pick the Ticket up again.
- Only `init`, `tickets` and `pickup` read the file, fresh from the clone's working tree.

### The server's config and folders

- **`$XDG_CONFIG_HOME/taiichi/server.toml`** holds `poll_interval` (seconds, default 60) and the Forgejo tokens under `[tokens]`, keyed by instance URL. URLs are compared after normalizing: lowercase scheme and host, no trailing slash. Because the file holds tokens, the server refuses to start if group or others can read it. Tokens can't come from environment variables.
- **The data folder** is `$XDG_DATA_HOME/taiichi` (by default `~/.local/share/taiichi`). It holds the database `taiichi.db` and the worktrees, at `worktrees/<job>`. It can't be moved separately; setting `XDG_DATA_HOME` moves it, which tests use.
- **The socket** is at `TAIICHI_SOCKET` if set, else `$XDG_RUNTIME_DIR/taiichi/taiichi.sock`, else `~/.local/state/taiichi/taiichi.sock`. The CLI and server work it out the same way, so the CLI never reads the server's config. The server creates the socket's folder with mode `0700`.
- **The CLI's own state:** a `submit` that fails keeps its text under `$XDG_STATE_HOME/taiichi/submit/` ([section 16](#16-cli)).

### How the server recognizes a Repo

- The CLI sends the clone's **Git common dir**, from `git rev-parse --path-format=absolute --git-common-dir`. Every worktree of a clone shares it, so the same answer comes back from the main checkout, from a Job's worktree, or from any worktree the user made. The server maps it to the Repo. The server's own Repo ID never leaves the server, and can't go in `taiichi.toml` because that file is shared.
- A server holds **at most one Repo per Forgejo repository** (instance URL plus `owner/repo`), since two clones of one repository would compete for the same Tickets.

### `taiichi init`, step by step

`taiichi init` opts a clone in. Running it again is always safe, and it's also how a moved clone is reconnected.

1. **Write `taiichi.toml` if it's missing.** The server writes it, so the format lives in one place, with every default spelled out. `forge.url` and `forge.repo` come from the `origin` remote when it's an HTTP(S) URL. Otherwise `--forge-url` and `--forge-repo` are required, and without them `init` fails with `422 forge_unknown`. `--ready-label` (repeatable) replaces the default `taiichi/ready`. An existing file is used as it is; passing these flags with one is `409 config_exists`.
2. **Check, in order,** stopping at the first failure:
   1. the file is valid (`409 config_invalid`);
   2. `server.toml` has a token for every Forgejo instance the file names (`409 token_missing`);
   3. Forgejo answers, giving the token's account (`GET /user`) and the repository with its default branch (`GET /repos/{o}/{r}`);
   4. every configured label exists.
3. **Offer to create labels.** `init` offers to create any missing position labels, and `taiichi/ready` if the ready labels are the default, all as exclusive labels in the `taiichi` scope. It asks first; `--yes` skips the question. Any other missing ready label is refused by name, because it's more likely a typo. `init` also warns, and asks, when a configured exclusive label shares its scope with exclusive labels that aren't in the config. In that case Forgejo would strip those other labels at every move. It names them and never refuses.
4. **Register the Repo,** which is one of three cases:
   - **Already registered** (the server knows this common dir): nothing changes, unless the file now names a different Forgejo repository, in which case the stored one is updated. A repository another Repo holds is refused with `409 forge_location_taken`.
   - **A moved clone** (the common dir is new, but a Repo for this Forgejo repository exists and its recorded folder is gone): the server updates the folder and runs `git worktree repair`, so active Jobs' worktrees work again. Moving or renaming a parent folder is the same case.
   - **New:** the server registers a new Repo. If a Repo for this Forgejo repository still exists at its recorded folder, `init` refuses.
5. **Report** the Repo, the Forgejo account Taiichi acts as, and a reminder to commit `taiichi.toml`. `init` never commits it.

> **Open:** the exclusive-scope warning in step 3 asks a question, but the API has no way to carry that question and its answer, the way it does for missing labels. See [Gaps found assembling the v1 spec](issues/22-gaps-found-assembling-the-spec.md).

### Removing a Repo by hand

v1 has no command for it. To remove a Repo:

1. Abandon its active Jobs.
2. Stop the server.
3. Run `DELETE FROM repos WHERE id = …`, which deletes its Jobs and their history too.
4. Delete those Jobs' worktrees under `$XDG_DATA_HOME/taiichi/worktrees/`.
5. Run `git worktree prune` in the clone.
6. Start the server.

## 13. Storage

Sources: [SQLite for the server's data](../../docs/adr/0001-sqlite-for-server-data.md), [Server internals](issues/12-server-internals.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md).

All of Taiichi's data lives in one SQLite database, `$XDG_DATA_HOME/taiichi/taiichi.db`, used through Ecto. Only the worktrees live on disk outside it. A transaction and unique indexes do the crash safety and cross-Job rules that files would have needed hand-written code for. PostgreSQL would have meant a second server to run, for features nobody needs.

**Settings,** all set explicitly: WAL journal mode; a busy timeout, so writers queue instead of failing; `synchronous = NORMAL`, so a power loss can lose the newest save but never corrupt the database; and foreign keys on. Migrations run at startup, each in its own transaction, and a failed migration stops the server.

Every table has an integer `id` with `AUTOINCREMENT`, so IDs are never reused.

| Table | Holds | Notable columns and constraints |
|---|---|---|
| `repos` | one row per registered Repo | `git_dir` (unique), main worktree folder, Forge type, URL and repo, Tracker type, URL and repo, `registered_at`; unique on (Forge URL, Forge repo) |
| `jobs` | one row per Job | `id` *is* the Job number people see. `repo_id`, Ticket ID, Tracker URL and repo, `settings` (JSON copy from Pickup), `branch`, `base_commit`, `worktree` (unique; null once removed), `picked_up_at`, `ended_at`, `ended_as` (`done`, `abandoned`, or null). A unique index on (Tracker URL, Tracker repo, Ticket ID) `WHERE ended_at IS NULL` allows one active Job per Ticket. |
| `attempts` | one row per Attempt | `job_id`, `seq` (order within the Job), `stage`, `number` (within the Stage), `started_at`; unique on (Job, `seq`) and on (Job, Stage, number) |
| `documents` | every document | `kind`, `meta` (JSON), `body`, `created_at`, and either `job_id` (the Ticket Snapshot) or `attempt_id`. An Attempt has at most one Stage Output. Feedback belongs to the Build Attempt it starts. `body` isn't loaded unless asked for, so listing stays cheap. |
| `input_parts` | each Attempt's Stage Input | `attempt_id`, `position`, `document_id`; written in the transaction that creates the Attempt |
| `shipments` | the Job's pull request, one row per Job | `url`, `number`, `ship_marker`, `last_report`, `reported_at`, `reports_failing_since`, `branch_head`, the new-comment count, the seen comment ID; created in Ship 1's save |
| `tickets` | the Ticket as the last check saw it, one row per Job | `state`, current title, body and other labels, `checked_at`, `checks_failing_since`, the seen point (`seen_at` plus the title, body and labels at the last `ack`, starting as the Snapshot at Pickup), `new_comments`; created in Pickup's save |
| `outbox` | steps after a save | `job_id`, `effect`, `payload` (JSON), `created_at`, `tries`, `last_error`, `last_tried_at`, `succeeded_at`, `given_up_at`; `id` gives the order |

**Worked out when read, never stored:**

- **The Position:** the current Attempt is the one with the highest `seq`. If it has no Stage Output, the Job is waiting for one; if it has one, the Job is at Ship waiting on the pull request. `ended_as` gives Done or Abandoned.
- **A Ship Attempt's Report** in the history: Refused if a Build Attempt follows it, Accepted if the Job is Done, otherwise the Shipment's `last_report`.
- **The current feedback cursor:** the one the latest Feedback moved to, or else the one in Ship 1's Manifest.
- "Differs from Snapshot" and "edited since seen".

A broken constraint becomes a domain error such as `ticket_has_active_job`, never a raw database error.

## 14. Inside the server

Sources: [Server internals](issues/12-server-internals.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), [HTTP over a Unix socket](issues/04-http-over-unix-socket.md), [Watching a Job live](issues/18-watching-a-job-live.md).

### Modules

The design follows domain-driven design: the domain rules sit apart from the HTTP layer and from storage.

```
Taiichi.Repos       the Repo: init, registration, lookup by Git common dir
Taiichi.Jobs        the Job: pickup, submit, abandon, ack, and every read
  Jobs.Pipeline     pure rules: Positions, moves, what goes in a Stage Input
  Jobs.Server       the process for one Job; the only writer of that Job
  Jobs.Checker      the checker for one Job
Taiichi.Outbox      the outbox sender
Taiichi.Tracker     the Tracker interface, and Tracker.Forgejo
Taiichi.Forge       the Forge interface, and Forge.Forgejo
Taiichi.Git         branches, worktrees, pushes
Taiichi.API         the HTTP router: JSON and error codes; calls only Repos and Jobs
```

- **Two aggregates.** An aggregate is a cluster of data changed only as a whole. The **Job** aggregate is a Job with its Attempts, documents, Stage Inputs, `shipments` row, `tickets` row and outbox rows. It changes only through its process, one transaction per change. The **Repo** aggregate is a registration and its Forgejo locations.
- **Ecto schemas are the domain model,** under one rule: only each module's store code calls `Repo`. `Jobs.Pipeline` is pure functions over schema structs that return them unsaved, so its tests build structs by hand with no database. Small functions per document kind read `meta` into Elixir values, such as an Evaluation's Verdict as an atom. A separate set of domain structs was rejected: it would have meant two structs per concept for little gain.
- **Ticket updates are a direct call.** After every save, the Job's process calls `Tracker.WriteBack.moved(old_job, new_job)`, which adds outbox rows to the save. v1 has no domain events. If a later feature needs to broadcast "Job 7 moved", it attaches at this call.

### Processes

- **One process per active Job,** a `GenServer`, with its checker beside it, under a `DynamicSupervisor`. The Job's process handles one write at a time for that Job: `submit`, `abandon`, `ack`, and moves the checker reports. Ended Jobs have neither process.
- **Reads go straight to the database:** `jobs`, `show`, `input`, `document` and `locate`. They never wait behind a slow push.
- **One outbox sender** for the whole server.

### Startup

1. Read `server.toml`, refusing to start if group or others can read it. Then look at the socket file. If there's no file, carry on. If connecting is refused, the file is left over from a crash: delete it. If connecting succeeds, another server is running: stop.
2. Open the database and run pending migrations.
3. Start a process and a checker for each Job that hasn't ended. Each checker checks at once.
4. Start the outbox sender. It first retries given-up rows, then sends unsent ones.
5. Open the socket, last, so no request sees a half-started server. Bandit listens with `ip: {:local, path}, port: 0`; it crashes without the `port: 0`.

On a Unix domain socket, Plug's `conn.remote_ip` is `{:local, ""}`, which Plug's own types don't expect. Nothing should rely on it.

## 15. HTTP API

Sources: [HTTP API shape](issues/10-http-api-shape.md), with changes from [Server internals](issues/12-server-internals.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md), and [The executor seam](issues/19-executor-seam.md).

### Conventions

- **Every call is `POST /v1/<method>`** with a JSON body, sent to `http://taiichi` over the socket. HTTP/1.1 needs a host name, and `taiichi` is a placeholder. This is plain remote procedure calls: reads use `POST` too, so there's one shape to learn. JSON-RPC 2.0 and a REST-style mix of `GET` and `POST` were rejected.
- **Every call is synchronous:** it answers only when its work is done, including Forgejo and Git work and the outbox's first try.
- **Versioning:** the server speaks only its own version. Any other prefix gets `404 unsupported_version` ("this server speaks v1; restart it"), since a long-running server can outlive a CLI upgrade.
- **Parameters are exact.** Every parameter is required, and ones that can be empty are sent as `null`. An unknown, missing or wrongly typed parameter is `400 bad_request`, naming it.
- **Formats:** keys in `snake_case`, timestamps in RFC 3339 UTC, and enum values as lowercase strings. Ticket IDs are strings (`"42"`), so later Trackers with IDs like `"ENG-42"` fit.
- **Stable shapes:** a result always has the same keys, with `null` for a field that doesn't apply. Every result is an object, and lists are wrapped (`{"jobs": [...]}`) so keys can be added later.
- **Data, not text:** the server returns structured data, and the CLI lays it out for people.

### Methods

Job numbers are unique across the server, so methods about one Job take just `job`.

| Method | Parameters | Result |
|---|---|---|
| `status` | `git_dir` (`null` outside a Git repository) | `{server: {version, api}, repo}`; `repo` is `null` if this clone isn't initialized |
| `init` | `git_dir`, `config`, `create_labels` | `{repo, account, wrote_config}` |
| `tickets` | `git_dir` | `{tickets: [{id, title, url, labels}]}` |
| `pickup` | `git_dir`, `ticket` | `{job}` |
| `jobs` | `git_dir`, `all` | `{jobs: [Job]}`, by number; active Jobs only unless `all` |
| `locate` | `worktree` | `{job}`, the number of the Job owning that worktree |
| `show` | `job` | `{job}`: the Job object plus `history` |
| `input` | `job` | `{stage, attempt, parts: [document]}` for the current Attempt |
| `document` | `job`, `stage`, `attempt` | `{document}`: that Attempt's Stage Output |
| `submit` | `job`, `stage`, `attempt`, `body`, `verdict`, `title`, `unjudged` | `{saved, job}` |
| `abandon` | `job`, `discard` | `{job}` |
| `ack` | `job` | `{job}` |

A Repo object, in `status` and `init`, is `{path, forge: {type, url, repo}, tracker: {type, url, repo}}`.

### Method details

- **`locate`** finds a Job from a worktree folder, matched exactly. The CLI calls it only when the user leaves the Job number out, passing `git rev-parse --show-toplevel`. Reading the number from the branch name was rejected: it would tie the CLI to the Forgejo Tracker's naming rule.
- **`submit`** carries the Stage and Attempt the text was written for, filled in by the CLI when the command starts. If the Job has moved on, the answer is `409 job_moved`, with the current `position` and `attempt` in `details`. If the Job has already submitted its Output and is waiting at Ship, it's `409 not_awaiting_output`. `verdict` and `title` are `null` where they don't apply, and `unjudged` is `false` except at Ship.
- **`abandon`** takes no Stage or Attempt ([section 7](#abandon)).
- **`document`** with `attempt: null` returns the latest Attempt of that Stage that has an Output. So `cat build` at Build 3 prints Build 2's notes.
- **`input`** still returns Ship's Input while the pull request is under review. On a Job that has ended it's `409 job_ended`, because there's no current Attempt; older Outputs stay readable through `document`.
- **`init`:** `config` holds only the settings the user typed, as JSON shaped like `taiichi.toml`, and is merged over the defaults when the server writes a new file. Anything left out, such as `forge.url`, comes from `origin`: the server reads it and finds the clone's main checkout itself. With no flags, the CLI sends `{}`. Missing labels stop the call with `409 labels_missing` (`details: {labels}`). The CLI asks the user, then calls again with `create_labels: true`; `--yes` sends `true` the first time.
- **`ack`** never calls Forgejo, so it never returns `502`.

### The Job object

Returned by `pickup`, `jobs`, `show`, `submit`, `abandon` and `ack`.

```json
{"number": 7,
 "ticket": {"id": "42", "title": "Add a dark mode toggle",
            "url": "https://forgejo.example/ngs/widgets/issues/42",
            "state": "open", "differs_from_snapshot": [], "edited_since_seen": [],
            "new_comments": 0, "checked_at": "2026-10-17T12:03:00Z", "checks_failing_since": null},
 "position": "ship", "attempt": 1, "awaiting": "shipment",
 "branch": "taiichi/42-j7-add-a-dark-mode-toggle",
 "worktree": "/home/ngs/.local/share/taiichi/worktrees/7",
 "shipment": {"url": "https://forgejo.example/ngs/widgets/pulls/51",
              "last_report": "pending", "reported_at": "2026-10-17T12:03:00Z",
              "reports_failing_since": null, "worktree_behind": false, "new_comments": 0},
 "outbox": null,
 "picked_up_at": "2026-10-17T09:12:00Z", "ended_at": null}
```

- **`position`** is `plan`, `build`, `evaluate`, `ship`, `done` or `abandoned`. `attempt` is `null` once the Job has ended.
- **`awaiting`** says who acts next: `stage_output` (the Operator must submit), `shipment` (waiting on review, including a Closed pull request), or `null` once ended. There's no "what's next" field; the CLI works it out from `position` and `awaiting`.
- **`ticket.state`** is `open`, `closed` or `deleted`. `differs_from_snapshot` and `edited_since_seen` list which parts changed (`title`, `body`, `labels`).
- **`shipment`** is `null` until Ship 1 is saved. `last_report` is `pending`, `refused`, `accepted` or `closed`. `worktree_behind` is the branch-head flag ([section 8](#what-it-looks-at-on-the-pull-request)).
- **`worktree`** is `null` once removed.
- **`outbox`** is `null` when nothing is pending, otherwise `{pending, failing_since, last_error}`, with given-up rows marked.

### History and documents

- **History,** on `show` only, has one entry per Attempt, oldest first. The current Attempt comes last, with `output: null`. Bodies aren't included; `document` fetches them.

  ```json
  {"stage": "ship", "attempt": 1, "started_at": "...",
   "output": {"kind": "manifest", "submitted_at": "...", "meta": {"shipment_url": "...", "commit": "c07b311"}},
   "last_report": "refused", "feedback_items": 2}
  ```

  `last_report` and `feedback_items` are `null` except on Ship Attempts.
- **A document** is `{kind, stage, attempt, meta, body}`. A Ticket Snapshot has `null` Stage and Attempt. Feedback carries the Ship Attempt it came from. A Stage Input comes back as an ordered list of documents, which a future AI executor can use without parsing text.

### Errors

An error body is `{"error": {"code": "...", "message": "...", "details": {...}}}`. The server writes `message` for people, and the CLI prints it as it is. Programs branch only on `code`. `details` appears only for codes that carry data.

| Status | Codes |
|---|---|
| 400 | `bad_request` |
| 404 | `unsupported_version`, `unknown_method`, `repo_not_initialized`, `no_such_job`, `no_such_ticket`, `no_such_document`, `not_a_job_worktree` |
| 405 (with `Allow: POST`), 413 (over Plug's 8 MB default), 415 | `method_not_allowed`, `too_large`, `unsupported_media_type` |
| 409: a state the user can fix | `job_moved`, `job_ended`, `not_awaiting_output`, `ticket_not_eligible`, `ticket_has_active_job`, `branch_taken`, `worktree_dirty`, `head_moved`, `branch_moved`, `shipment_closed`, `shipment_accepted`, `forge_location_changed`, `forge_location_taken`, `labels_missing`, `config_invalid`, `token_missing`, `config_exists` |
| 422: content that's invalid | `invalid_output` (an empty body, a Verdict missing or not allowed, an empty title), `forge_unknown` |
| 500 | `internal` (anything else, including a Git failure) |
| 502 | `forgejo_failed` (always means nothing was saved) |

404 also covers anything named in a request that the server doesn't have, and a Forgejo failure is 502. That follows the common reading people expect when they read `curl -v` output and logs. New codes are added as needed.

## 16. CLI

Sources: [CLI commands](issues/06-cli-commands.md), [CLI language](issues/11-cli-language.md), with changes from [`taiichi init` and configuration](issues/09-init-and-configuration.md), [Server internals](issues/12-server-internals.md), [An operation stopping partway](issues/15-operation-stopping-partway.md), [A Shipment changing under a Job](issues/16-shipment-changing-under-a-job.md), and [A Ticket changing under a running Job](issues/17-ticket-changing-under-a-job.md).

### Commands

Each command is a single verb. A Job's number goes last. Inside a Job's worktree it can be left out, and the CLI asks the server which Job owns the folder (`locate`). A number given explicitly always wins, and it works from any folder.

```
taiichi status                         is the server reachable? is this clone a Repo?
taiichi init [--forge-url URL --forge-repo OWNER/REPO]
             [--ready-label LABEL]...  write taiichi.toml and register the Repo
             [--yes]                   create missing labels without asking
taiichi tickets                        Tickets ready for Pickup
taiichi pickup <ticket>                start a Job; prints its number, branch and worktree
taiichi jobs [--all]                   this Repo's active Jobs (--all adds ended ones)
taiichi show [job]                     Position, branch, worktree, pull request, Ticket, history, what's next
taiichi input [job]                    the current Attempt's Stage Input
taiichi path [job]                     the worktree folder, and nothing else
taiichi cat <stage> [job] [-n N]       one stored Stage Output (default: the latest)
taiichi submit [job] [-f FILE | -]     submit the current Stage's Output
        [--pass | --fail]                Evaluate only; exactly one is required
        [--title TITLE]                  Ship only; optional
        [--unjudged]                     Ship only; ship a head Evaluate didn't judge
taiichi ack [job]                      mark what's new on the Job as seen
taiichi abandon [job] [--yes] [--discard]
                                       end the Job as Abandoned (asks first)
```

`path` reads the worktree from `show`, and `cat` calls `document`. Left out of v1: a `history` command (`show` prints history), `--json` output (planned soon after v1), following a Job live, and a "check now" command.

### Behaviour

- **Every command** runs `git rev-parse` to find the Git common dir, plus `--show-toplevel` and a `locate` call when the Job number is left out. Then it posts JSON over the socket. Error messages come from the server; the CLI only branches on error codes.
- **Submitting:** one `submit` covers every Stage.
  - The CLI checks `--pass`/`--fail` against the Job's Stage before opening the editor, so a missing Verdict fails before any writing is lost.
  - The text comes from `-f FILE`, from stdin with `-`, or from `$EDITOR`. The editor opens the way `git commit -v` does: the Stage Input appears below a scissors line, and everything from that line down is cut before sending. An empty text aborts.
  - If the Input includes the Ticket Snapshot and the Ticket has changed since, a note sits above the scissors line, such as "Ticket #42 has changed since this Snapshot: body. See <url>." The stored Input doesn't change.
- **A failed submission** whose text came from the editor or stdin is saved to `$XDG_STATE_HOME/taiichi/submit/<job>-<stage>-<attempt>.md`, and the CLI prints the command to retry: `taiichi submit <job> -f <path>`.
- **`show`** prints the Position, the Ticket's block, the pull request and its flags, any outbox warning, the history, and what to do next. The Ticket's block looks like this:

  ```
  Ticket #42 · closed · checked 12:04
    differs from Snapshot: body
    edited since seen: body
    2 new comments
  ```

- **`jobs`** marks a Job that needs attention: the Ticket closed or deleted, the Ticket edited since seen, new comments on the Ticket or pull request, the pull request Closed, the worktree missing commits from the branch, failing checks, or pending or failing outbox rows.
- **`init`** makes its second call with `create_labels: true` after the user agrees.
- **Prompts** first check for a terminal, and without one fail with a message naming `--yes`.

### Libraries

Go with the Charm libraries: Cobra for commands and shell completion, Fang to style Cobra's help and errors, Huh for prompts, Lip Gloss for tables and styling, and Glamour for Markdown. TypeScript on Bun was the runner-up. The choice gets a second look before the post-v1 TUI is built.

From the prototypes, for whoever builds it:

- Huh opens `/dev/tty` even when stdin is redirected, hence the terminal check before every prompt.
- Fang asks the terminal for its background colour and stalls about 4 seconds on a terminal that never answers. Use a short timeout, or set the colour scheme explicitly.
- Glamour v2 doesn't detect the background colour, so detect it ourselves. Print Markdown unstyled when stdout isn't a terminal.
- Huh's standalone forms always render light themes (huh#834). The prototype has a 6-line fix.
- Find the editor with `git var GIT_EDITOR`.
- The Go prototype is on branch `prototype/cli-stacks`, in `cli/prototype-go/`, with a `NOTES.md`.

## 17. Not in v1

From the map's [Out of scope](map.md#out-of-scope) section and [The executor seam](issues/19-executor-seam.md):

- **Other Trackers and Forges:** GitHub, GitLab, Linear, the repo-local `.tracker/` files, and a manual Forge. The interfaces stay general enough for them.
- **AI doing Stages.** An AI executor would use `show`, `input`, the worktree and `submit` as they stand. Every AI decision waits for a session after v1. The open items: an `awaiting` field on `input`, how an executor learns a Stage Input is ready, recording who wrote an Output, and the word for whatever writes Stage Outputs.
- **A CLI setup wizard for `init`:** [CLI init wizard](../taiichi-post-v1/issues/01-cli-init-wizard.md).
- **Listing and removing Repos:** [List and remove Repos](../taiichi-post-v1/issues/02-list-and-remove-repos.md).
- **`--json` output** from the CLI, soon after v1.
- **Gates,** a human check that can send a Stage Output back: [Gates](../taiichi-post-v1/issues/04-gates.md).
- **Following a Job live:** [Follow a Job live](../taiichi-post-v1/issues/05-follow-a-job-live.md). v1 reserves nothing for it.
- **A terminal UI,** after [Revisit the CLI language before the TUI](../taiichi-post-v1/issues/03-revisit-cli-language-before-tui.md).
- **Remote deployment.**

## Readings made while assembling

The tickets imply these without stating them. Each is written into the spec as read here; correct any that read wrong.

1. **The worktree folder is recorded at Pickup's save** (`worktrees/<job>`), so `pickup` returns it while the outbox step that creates it may still be pending. The Job's `outbox` field shows whether it is.
2. **The `tickets` row is written in Pickup's save,** since its seen point starts as the Snapshot and Pickup's time.
3. **Refused moves only a Job waiting at Ship.** At Ship before the Manifest is in, as away from Ship, new reviews are older than the next Ship marker and count for nothing.
4. **The list of outbox steps** in [section 9](#9-the-outbox) adds Ship's steps (the description, the comment, re-requesting review) to the examples in [An operation stopping partway](issues/15-operation-stopping-partway.md). It also folds creating the branch into adding the worktree.
5. **Outbox rows need a `skipped` ending,** which the column list doesn't show. A `skipped_at` column, or one result column, is the implementer's choice.
6. **Some Job object field names.** `ticket` merges the first version's `{id, title, url}` with the fields added later. The pull request's branch-head flag and new-comment count were never named; this spec calls them `worktree_behind` and `new_comments`. `differs_from_snapshot` and `edited_since_seen` are lists of what changed, matching `show`'s output. `ack` replies with the Job object, like the other Job writes.
7. **The feedback cursor may need two IDs.** Feedback gathers review bodies (review IDs) and comments (comment IDs), so the Forgejo Forge decides the cursor's exact shape inside `meta`.
8. **The labels-exist check also runs on `tickets` and `pickup`,** not only `init`, answering `409 labels_missing`, because [Forgejo Tracker rules](issues/07-forgejo-tracker-rules.md) has the server fail clearly on a missing label.
9. **Left to the implementer:** the slug's length, the wording of each comment Taiichi posts, and reading `server.toml` once at startup (implied by a fixed token taking effect on restart).
