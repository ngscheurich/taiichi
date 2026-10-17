# CLI commands

Type: prototype
Status: resolved

## Question

What commands does the CLI offer, with what arguments and output? Prototype it as a rough command reference with example sessions, covering at least: `taiichi init`, Pickup, listing and showing Jobs, printing a Job's worktree path, submitting a Stage Output, abandoning a Job, and anything a human needs to drive a Job from Pickup to Done.

## Answer

Settled with the user against a prototype of three grammars (flat verbs, noun groups, and one verb per Stage inside the worktree), each run through the same Job from Pickup to Done.

**Grammar: flat verbs.** Each command is one verb, and Job commands take the Job as their first argument.

**Naming a Job**
- Jobs are numbered per repo (1, 2, 3). The CLI already picks the repo from the current directory, so the number only has to be unique within that repo. It also keeps Job and Ticket apart: Job 7 of Ticket #42.
- Inside a Job's worktree, the Job argument may be left out. The CLI asks the server which Job owns the current directory. An explicit Job argument always wins, and outside a worktree the argument is required.

**Commands**

```
taiichi status                        server reachable? this repo initialized?
taiichi init                          write the config file, register the repo
taiichi tickets                       Tickets eligible for Pickup
taiichi pickup <ticket>               start a Job; prints its number, branch, worktree
taiichi jobs [--all]                  this repo's active Jobs (--all: Done and Abandoned too)
taiichi show [job]                    position, branch, worktree, Shipment, history, what's next
taiichi input [job]                   the current Attempt's Stage Input
taiichi path [job]                    the worktree path, nothing else
taiichi cat [job] <stage> [<n>]       one stored document (default: latest Attempt)
taiichi submit [job] [-f FILE | -]    the current Stage's Output
        [--pass | --fail]               Evaluate only, exactly one required
        [--title TITLE]                 Ship only, optional
taiichi abandon [job]                 end the Job as Abandoned (asks first; --yes skips)
```

**Submitting**
- One `submit` covers every Stage. It was reconsidered with the full verb list in view and kept over one verb per Stage (named for the Stages or for their Stage Outputs). `show` and the error messages say what the Job expects next.
- The Verdict is given with `--pass` or `--fail`: exactly one is required at Evaluate, and both are refused at other Stages. The CLI checks the Job's Stage before opening the editor, so a missing Verdict fails before any writing is lost.
- The body comes from `-f FILE` or from stdin (`-`). With neither, `$EDITOR` opens as `git commit -v` does: the Stage Input sits below a scissors line, which is cut before sending. An empty body aborts. The CLI always sends the contents, never a path.

**Left out**
- `history`: `show` already prints a Job's history, and `cat` prints any one stored document.
- `--json` output: not in v1, but planned as a fast-follow after v1. Meanwhile scripts can call the server's HTTP API.
- Following a Job live is still fog on the map. There's no manual "check the Shipment now" command, since the server polls the Forge on its own.

Prototype: `docs/prototypes/cli-commands.md` on branch `prototype/cli-commands` (commit `cb2e2b3`). The branch is local only: the push failed because the sandbox has no `ssh`.

No glossary change: no domain terms moved.

## Comments

**From [`taiichi init` and configuration](09-init-and-configuration.md):** `taiichi init` takes `--forge-url` and `--forge-repo` (required when `origin` isn't an HTTP(S) URL), a repeatable `--ready-label`, and `--yes` to skip the offer to create missing labels. Running it again is safe, and it also relinks a moved clone. `taiichi repos` was considered and moved post-v1.
