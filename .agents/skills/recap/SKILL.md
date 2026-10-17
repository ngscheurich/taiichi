---
name: recap
description: Generate a commit-by-commit recap of unpushed work or a chosen commit range, then wait for annotation. Use when the user asks to walk through, narrate, explain, or review their unpushed work, recent commits, or a chosen range of commits. Not for explaining how the system works from the source — that is the `tour` skill.
compatibility: Requires git; reads local history, so the range reflects the last fetch
---

# Recap a set of commits

Generate a commit-by-commit Markdown recap of a range of local commits — by default everything un-pushed, or a range the invoker chooses (see step 1) in `.scratch/recaps/` — covering overall structure, notable code, and decisions with justification, then wait for annotation.

## Audience

Don't assume a fixed reader profile. Decide the depth of explanation from these rules, in order:

1. **The `audience:` arg, if the invoker passed one** (e.g. `audience: backend engineer new to both`). An explicit arg wins outright.
2. **Otherwise, your memory of the user's skillset** — the languages they're fluent in, _and the specific idioms you've already explained to them in past recaps and tours._ Explain a construct the first time it appears for this user; once memory records it as known, reference it plainly and don't explain it again. This is what stops the same idioms (closures, generics, error-handling idioms, …) being explained run after run as the user's fluency grows — step 5 of the Steps list keeps the record current.
3. **If you have neither an arg nor a relevant memory, treat every language as unfamiliar:** explain any genuinely non-obvious construct whatever its language, and state in the preamble that the audience was unspecified.

## Steps

1. **Determine which commits to cover.** The result is a list of shas — a contiguous range or, for a selector, a possibly non-contiguous set.
   - **If the invoker named the commits (e.g. a `select:` arg), use that.** Forms:
     - A git range expression, used verbatim: `HEAD~3..HEAD`, `main..HEAD`, `abc123..def456`.
     - A bare integer `N` — the last `N` commits (`HEAD~N..HEAD`).
     - A single revision — that commit alone (`<rev>^..<rev>`).
     - A natural-language selector — translate it to `git log` filters, scoped to the unpushed range by default (widen if the phrasing reaches further back, ask if the scope is unclear). E.g. `select: commits with a Co-Authored-By trailer` → `git log --grep='Co-Authored-By:' --format=%H @{u}..HEAD`. Other filters: `--author`, `--since`/`--until`, a trailing `-- <pathspec>`.
   - **Otherwise, default to the unpushed range.**
     - If the branch has an upstream (`git rev-parse --abbrev-ref @{u}` succeeds), it is `@{u}..HEAD` — commits not yet on the remote-tracking branch.
     - Otherwise fall back to the trunk: try `origin/main..HEAD`, then `main..HEAD`.
   - List the resolved commits to confirm — `git log --oneline <range>`, or the filter command for a selector. If empty, stop and say so: for the default, that nothing is unpushed; for a named set, that it matched nothing (bounds reversed, already merged, or no commit matches).

2. **Read each commit, oldest first.** For every SHA run `git show --stat <sha>` for the message and file list, then `git show <sha>` for the diff. When a hunk needs context, open the file with the read tools rather than guessing.

3. **Write the recap** to a file: `name=$(git branch --show-current | tr / -); out=".scratch/recaps/${name:-$(git rev-parse --short HEAD)}-recap.md"` (the fallback covers a detached HEAD, where the branch name is empty). Resolve naming conflicts by appending a `-<number>` to the filename. Follow the structure below and cover the commits in order so the narrative builds.

4. **Wait for review** then respond to the annotations the user sends back.

5. **Record what you explained** (only when the audience came from audience rule 2 or 3 — not from an `audience:` arg, which frames a one-off reader and says nothing about the user). Append every idiom you explained for the first time this run to the user's skillset memory, so the next recap or tour references it plainly instead of explaining it again. If an annotation shows the user already knew something you explained, record that too. This is how the record gets built from real evidence rather than a guess.

## recap structure

- **Top-level heading (`#`)** — a single descriptive H1 that names the work, not a generic "recap." Prefer the theme of the branch, e.g. `# Auth server scaffold` over the branch slug. Follow it with a one-line preamble: branch, the commits covered (range or selector), and the audience it's written for.
- **Summary** — before any commit sections, a short prose overview (a paragraph or a few bullets) of what the work in the range accomplishes as a whole and how the commits relate. The reader should grasp the shape of the change here without scrolling further.
- **One `##` section per commit, in order.** The header must name the commit unambiguously: ``## 1. <subject> (`<short-sha>`)``.
  - _What it does_ — the change in one to three sentences.
  - _Structure_ — files touched and how they fit together; reference code as `path:line`.
  - _Notable code_ — quote the genuinely interesting hunks and explain the constructs the reader wouldn't already know. Skip boilerplate, generated, and formatting-only diffs.
  - _Decisions_ — bullet each non-obvious choice with its justification ("chose X over Y because …"). Mine the commit body for rationale before inventing your own.
- **Closing** — themes, follow-ups, or risks that span more than one commit.

## Notes

- Do not run `git fetch` unless asked; the range reflects the last fetch.
- When a selector yields non-contiguous commits, still narrate them oldest-first and say in the preamble that they're a filtered set, not a continuous range.
- Highlight, don't dump — a recap is curated narration, not a re-printed diff.
- `.scratch/` is a working directory for artifacts that aren't part of the project's own documentation. Create it if absent, and add it to the project's ignore file if it isn't already covered.
