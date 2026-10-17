---
name: sweep-comments
description: A procedure for evaluating and trimming code comments across the codebase. Use when the user says "sweep comments", "check for comment quality", or otherwise wants an agent to clean up code comments.
compatibility: Requires git for the comment-only verification gate; the project's own formatter, parser, or linter strengthens it
---

# Run a comment sweep

> [!IMPORTANT]
> The _rules_ live in the project's prose style guide; this is the _process_ that applies them at scale without drifting off the bar. When the project has no such guide, the `write-prose` skill resolves the fallback.

A sweep cuts comments that don't earn their place. The hard part is not knowing the rules — it's holding a consistent, aggressive altitude across dozens of files. This guide exists because the rules alone can underdetermine that altitude and settle too timid.

---

## 1. The core principle

A comment exists to protect the next editor from breaking something **they cannot see in the code**. That is the whole job. Apply one litmus to every comment:

> **Would the next editor break something they can't see from the code? If no, the comment should not exist.**

Apply it strictly. "It helps you skim," "it's a nice summary," "it's conceptually relevant" — none of these pass. Cut harder than feels safe; the instinct to keep is almost always too generous.

---

## 2. The per-comment decision procedure

Walk every comment through this. Most land on "delete."

1. **Does it document an external tool's behavior?** (What the language runtime, the OS, the version control system, a library, or a separate service does.) → Rewrite to state _our_ choice and its consequence, or delete. _"Without `IS_SANDBOX=1` the hands-off launch exits at startup"_ — not a paragraph on why the external tool refuses to run as root.

2. **Does the function body show it within a few lines?** → Delete. The flags a branch passes, the default a value resolves to, "returns nothing on failure" sitting directly above the branch that returns nothing on failure — all visible, all cut.

3. **Is the item private / unexported?** → Default to **delete**. Keep a one-line comment _only_ for a genuine gotcha the code cannot show: an ordering dependency, a deliberate absence, a footgun an editor would trip over. **Never describe what a private function does** — the name carries that. If you're restating the name, delete.

4. **Is the comment there to translate a name?** (A unit, what it returns, what the constructor means.) → The name is wrong. Note it for a rename pass; delete the comment. Don't rename mid-sweep — see §6.

5. **Does it cite a doc/ADR, recount history, or describe the roadmap?** → Delete those parts. No `docs/adrs/…`, no bare `ADR-0007`, no "this once dropped…", no "retired X," no "will become."

### A worked anchor

Before, on a private backoff helper (the comment token here is illustrative — use the language's own):

```
// The stages this walks, in order: the base delay doubled per attempt, then
// clamped to the ceiling from config, then jittered. Note we clamp before
// jitter rather than after — the retry ADR has the full reasoning. It used to
// clamp after jitter; that changed in the 2.0 rewrite. Note also that the HTTP
// client retries idempotent requests once on its own; that layer is separate
// and will eventually be folded into this one.
```

After:

```
// Clamp before jitter, not after: clamping last pins every retry to exactly
// the ceiling and destroys the spread.
```

Two lines survive, and only because they stop an editor from "tidying" the order and silently collapsing the jitter. "In order," the doubling, and the config ceiling are all visible in the body; the ADR citation and the rewrite history are noise; the HTTP client's own retry is an external tool's behavior, and folding it in is roadmap.

---

## 3. Hard constraints

- **Only touch comments.** Code tokens stay byte-identical. (Formatters may re-align whitespace as a _consequence_ of removing a trailing comment — that's fine; tokens are unchanged.)
- **Never modify runtime string literals** — user-facing messages, help text, embedded shell scripts, protocol text — even when they contain an ADR reference. They are data, not prose.
- **Don't rename in the sweep.** A rename is a code change with its own blast radius. Record bad names; do them as a separate follow-up pass (§6).

---

## 4. The workflow

### Pin the comment syntax first

Every command below is parameterized by the language's comment tokens. Establish them before running anything, and record them in the batch briefs:

- **Line comment** — `//` (C, Java, Go, Rust, TypeScript, Gleam), `#` (Python, Ruby, shell, Elixir, Perl), `--` (SQL, Haskell, Lua, Ada), `;` (Lisp, Clojure), `%` (Erlang, LaTeX).
- **Block comment**, where the language has one — `/* */`, `<!-- -->`, `=begin`/`=end`, `{- -}`.
- **Doc comment**, where it is distinct from the line comment — `///`, `/** */`, a leading string literal, an attribute or annotation.

A codebase with several languages is several sweeps, one per syntax. Don't carry one language's tokens — or its doc conventions — into another.

### Scope

Rank files by comment volume and start with the worst offenders: that is where the bar matters most and where calibration pays off.

```sh
lc='//'        # the line-comment token pinned above
ext='ts'

find <src> -name "*.$ext" -print0 | while IFS= read -r -d '' f; do
  printf '%s %s\n' "$(grep -cF "$lc" "$f")" "$f"
done | sort -rn
```

This is a **ranking heuristic, not a measurement**. It counts lines containing the token anywhere, so it catches trailing comments but also counts the token inside string literals and URLs, and it misses block comments entirely. That is fine for choosing what to open first; never quote it as a result.

### Calibrate by hand first

Sweep the single worst offender yourself, before delegating anything. Get the user to sign off on that file. This is not optional — it converts the abstract bar into a concrete altitude everything else measures against. Expect to re-cut it two or three times as the user pushes you lower; the corrections _are_ the calibration. Commit the approved file as a reference.

### Batch and delegate

Group the remaining files into batches (by subsystem reads well). For each batch, hand each file-group to a subagent with: the rules (the §-references above), the pinned comment syntax, the _committed reference files_ (one aggressive, one moderate — they show the target altitude better than prose), and the hard constraints. Tell each subagent to **edit and format only — never build, test, or run the project's gates**, because concurrent runs race. Have it report per-file before/after counts and flag any borderline call.

### Gate every batch

Before committing a batch, run all three gates from the parent (not the subagents):

1. **Static gate** — whatever the project has that would catch a broken file: compile, typecheck, or lint. If it has none, say so explicitly rather than skipping silently.
2. **Test** — the full suite passes.
3. **Comment-only proof** — evidence that no code token moved.

For the third gate, use the strongest option the language supports, in this order.

**(a) Compare a token or syntax stream.** Best by far, and the only option that is actually sound. If the language ships a formatter that can drop comments, an AST or token dump, or a parser you can drive (`tree-sitter parse`, the compiler's own dump flags), run it over both revisions and diff the output. Comments vanish from both sides by construction, and nothing inside a string literal can be mistaken for one.

**(b) Read every hunk.** `git diff -U0` or `git diff --word-diff` over the batch, confirming by eye that each hunk is entirely comment. Slow but reliable, and the right fallback when no tool from (a) exists.

**(c) Strip line comments by regex.** Last resort, and **it can produce a false all-clear** — see the caveats below.

```sh
bash -c '
lc="//"    # the line-comment token pinned above
norm() { sed -E "s:${lc}.*\$::" | tr -s "[:space:]" " " | grep -vE "^ *\$"; }
for f in $(git diff --name-only -- "<glob>"); do
  d=$(diff <(git show HEAD:"$f" | norm) <(norm < "$f"))
  [ -n "$d" ] && { echo "VERIFY BY HAND: $f"; echo "$d"; }
done
echo done'
```

Two caveats make this tier weak, and both need stating in the batch's commit message if you rely on it:

- **String literals mask real changes.** The regex truncates at the first token occurrence on a line, whether or not it is a comment. A line reading `url = "http://host/a"` normalizes to `url = "http:` on both sides, so editing that path to `/b` passes the gate silently. Before trusting a clean result, `grep -F "$lc"` the batch for the token inside string literals and read those lines by hand.
- **Block comments are not stripped.** Removing one shows up as a difference, so expect hunks and verify each. The gate flags files for review; it does not clear them.

If a gate fails, fix before committing. Commit per batch with a message that records what was cut and how the code was verified unchanged.

### Checkpoint with the user

Decide up front, with the user, how much review they want per batch — full-diff or spot-check — and whether to keep delegating. Surface genuine policy forks (e.g. "do self-evident variant docs keep a one-liner?") as questions rather than guessing; the answer applies to every remaining file, so resolving it once prevents rework. When a ruling lands, **fold it back into the style guide and your memory** so the next sweep starts already calibrated.

---

## 5. Pitfalls

- **Don't chase the comment-line count.** Doc comments, trailing blank lines, and per-member docs all inflate it, so a file that cut a long essay down to a couple of lines may show a small delta. Judge by what's left, not by the number. A file that was mostly legitimate public API docs should barely move — that is the _right_ result, not timidity — while a file documenting an external tool at length should collapse.
- **Formatter realignment is benign.** Removing a trailing field comment lets the formatter re-align the surrounding block. The verification in §4 ignores whitespace, so it won't flag this — but a naive line diff will. Verify tokens, not columns.
- **Shell word-splitting bites.** A `files="a b c"; for f in $files` loop silently runs once on the whole string in `fish` and other non-POSIX shells, producing a false "all clear." Always run verification loops through `bash -c`.
- **Doc-comment conventions are per-language and usually unwritten.** They differ on trailing blank lines, on whether the identifier name leads the sentence, and on whether each variant or field carries its own entry. Read the project's existing public API docs before editing them, and never carry one language's form into another.
- **The merge after a long sweep.** If the base branch moved, a conflict in a swept file is usually "their new code + your comment treatment." Take their functionality, apply your trimming to it — don't re-sweep their new code wholesale inside a conflict resolution.

---

## 6. Follow-up: names the comments were propping up

During the sweep you'll find comments that exist only because a name is weak (a `callError` that actually _shapes_ an error; a passthrough wrapper that adds nothing). Don't fix these mid-sweep. Afterward, run a small, separate refactor pass: rename so the name carries what the comment did, then delete the now-redundant comment. Keep it scoped, gated by the project's build and tests, and committed apart from the sweep — it's a code change, not a comment change.
