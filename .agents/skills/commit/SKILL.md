---
name: commit
description: Write and create Git commits in house style — Conventional Commits plus a standard prose convention. Use when committing changes, drafting or amending a commit message, or whenever the user asks you to commit.
compatibility: Requires git
---

# Write and create a commit

Write every commit as a **Conventional Commit** that also follows a standard prose convention. Shape:

```
type(scope): summary

Optional body explaining why, hard-wrapped at 72 columns.

Co-Authored-By: <agent name and version> <noreply address>
```

## Subject line

- `type(scope): summary` — include the scope when the change is local to one area.
- Imperative mood ("add", "drop", "correct"), lowercase after the colon, no trailing period.
- Keep it concise and under 72 characters.
- Describe the change and its intent, not the files touched.

## Types

Pick the type that names the change's intent, not the one matching the largest part of the diff.

- `feat` — a new user-facing capability
- `fix` — a bug fix
- `docs` — documentation only
- `style` — formatting/whitespace, no change in meaning or behavior
- `refactor` — a code change that neither adds a feature nor fixes a bug
- `perf` — a change that improves performance
- `test` — adding or correcting tests
- `build` — build system, tooling, or dependencies
- `ci` — continuous integration config or pipelines
- `chore` — maintenance that touches neither src nor tests
- `revert` — reverts a previous commit

## Body

Include a body whenever the change isn't self-evident from the subject. It explains the **why** — the reasoning, the constraint, the tradeoff — not the what the diff already shows.

- Separate it from the subject with one blank line, and hard-wrap at 72 columns.
- A body that walks area by area through what changed has become a diff summary. The diff is already in the commit — cut the tour and keep the reasoning.
- Reference the decision record or acceptance criteria that motivated the change (e.g. `ADR-0011`, `AC #4`) when one applies.
- Genuinely trivial commits (a one-line chore, a typo fix) need no description in the body, but it should still contain an attribution trailer (see below).

## Make the commit

The message is half the job. These steps decide what the commit actually contains.

### Scope it to one change

A commit records one logical change. When the working tree holds several — a fix plus an unrelated rename, a feature plus a drive-by cleanup — split them into separate commits rather than one mixed commit. A reader reverting a bug fix should not lose an unrelated refactor with it. If the work genuinely cannot be separated, say so in the body rather than pretending it is one thing.

### Read the diff before writing the message

Run `git status` and `git diff` (plus `git diff --staged` when anything is already staged) and read them. Write the message from what the diff shows, never from your memory of what you set out to do — the two diverge more often than they don't, and the diff is what ships.

### Stage deliberately

- Stage explicit pathspecs. Reach for `git add -A` or `commit -a` only after reading `git status`, and never when it would sweep in files you did not touch.
- Check the untracked list before staging broadly. Build output, editor scratch files, credentials, and local config get committed by accident this way; if something untracked should never be committed, add it to the project's ignore file instead.
- Confirm the branch is the one you mean to commit to (`git branch --show-current`). Where a project has a policy about which branches take direct commits, follow the project's, not a default.

### When a hook rejects the commit

A failing pre-commit hook is a finding, not an obstacle. Read its output and fix the cause. Do not reach for `--no-verify` — if a hook genuinely must be bypassed, that is the user's call to make, so ask.

A hook that reformats files leaves the fix unstaged, so the commit records the unformatted version. Re-stage and re-read `git diff --staged` before retrying.

### Keep the repo's signing behavior

If the surrounding commits are signed, yours must be too — `git log --format='%h %G?'` shows this. When signing fails, stop and report it rather than committing unsigned; an unsigned commit in a signed history is a change to the repo's guarantees, not a detail.

### Verify what landed

After committing, confirm the commit holds what you intended and nothing more: `git show --stat HEAD` for the file list, `git status` to see what remains uncommitted. Report anything left behind rather than leaving the user to discover it.

## Amending and rewriting

Amend freely while a commit is local. Once it is pushed it is shared history, and changing it means a force-push — propose the replacement message and let the user decide. `git log --format='%h %s' @{u}..HEAD` shows which commits are still yours to rewrite.

## Breaking changes

Mark a breaking change with `!` before the colon (`feat(server)!: ...`) and add a `BREAKING CHANGE:` footer describing the break.

## Prose conventions

A commit message is prose and should be written in a style cohesive with the project voice; use the `write-prose` skill when composing the message.

## Attribution

End every commit you author with a `Co-Authored-By` trailer, after a blank line, that identifies you — the authoring coding agent — by name. Fill in your own identity:

```
Co-Authored-By: <agent name and version>
```

Do not prepend the model name with the company name:

- **Good:** `Claude Fable 5.1`
- **Bad:** `Anthropic Claude Fable 5.1`, `Anthropic: Claude Fable 5.1`

### Email address

If applicable, the following email addresses should be listed in angle brackets after the model name:

- **Anthropic models:** `noreply@anthropic.com`
- **OpenAI models:** `noreply@openai.com`

Other models should omit the email address.

Examples:

```
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
```

```
Co-Authored-By: GPT 5.6 Sol <noreply@openai.com>
```

## Examples

```
feat(config): read the frontmatter map without parsing the body

Only the scalars and flat lists in the frontmatter are parsed and the
body stays opaque, per ADR-0011, so a document whose body is malformed
still yields its metadata. Absent fields read as absent rather than
erroring, so partial documents don't crash reads.

Co-Authored-By: Example Agent 1.0 <noreply@example.com>
```

```
chore: drop the stale dependency lockfile

Co-Authored-By: Some Agent
```

## Signing

If project commits are signed, but you don't have access to a signing mechanism, proceed with unsigned commits; it is the user's responsibility to sign them before pushing to a Git forge.
