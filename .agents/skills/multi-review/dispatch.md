# Dispatching reviewers and the judge

How to hand the briefs to the agents that execute them. Pick the variant matching your executor. The rules that hold in both variants:

- You are the coordinator, never a reviewer and never the judge. Do not execute a brief yourself, and do not write any review or judgement file — one authored by you is invalid, because you also write the synthesis.
- Every reviewer receives the identical brief.
- A phase is done only when its output files exist; verify before reading.

## With subagents

Launch all reviewer briefs in a **single message with N agent calls** so they run concurrently. Then spawn one judge subagent and hand it all reviews verbatim.

## Without subagents

**Reviewers.** Write each brief to a temp file (`mktemp`) after appending:

> Write your findings to a file: `.scratch/reviews/<slug>/<uuid>.md`.

Dispatch each brief to the agent executor available in this environment and wait. When you resume, verify with a directory listing of `.scratch/reviews/<slug>/` (or `test -f` on each path) before reading the reviews; if any file is missing or empty, report which and ask the user to re-dispatch rather than writing it yourself.

**Judge.** Write the judge brief to a temp file after appending:

> Write your findings to a file: `.scratch/reviews/<slug>/judgement.md`.

Dispatch this brief and wait. When you resume, verify `judgement.md` exists the same way; if it is missing or empty, report that and stop. Your only authored output is the synthesis in step 5.

**If no executor is available**, stop and tell the user instead of filling the reviews or the judgement in.
