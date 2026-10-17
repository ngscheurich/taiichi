---
name: tour
description: Author a comprehensive, top-to-bottom developer tour of the whole project or one subsystem, built from the source — terminology first, broad strokes then every rabbit hole, verbatim code labelled with file:line, and Mermaid diagrams for processes. Use when the user asks to deeply explain, document, or onboard someone to the architecture of the project or a subsystem from the code itself. Not for narrating a range of commits — that is the `recap` skill.
compatibility: Requires git and an agent runtime that can dispatch subagents in parallel
---

# Create a codebase tour

Produce one long-form Markdown document that guides a developer through a subject end to end: what it is, the terminology, the shape, then every load-bearing detail, showing real code and diagramming real flows. The output is durable onboarding documentation built from the source, not a narration of recent changes.

## Scope it first

The user may name a specific subsystem to tour; otherwise the scope is the whole project.

- **Whole project** — the orientation doc a new contributor reads once. Runs big-picture → cross-cutting foundations → each subsystem → appendices.
- **One subsystem** — a focused deep-dive (the transport layer, the agent runtime). Same rules, narrower blast radius.

Name the file for the scope: `docs/tours/<scope>.md` — `docs/tours/project.md` for a whole-project tour, `docs/tours/agent-runtime.md` for a subsystem. Follow the project's own documentation layout where it has one. A tour is durable documentation and gets committed (step 5), so it does not belong in a scratch directory. Resolve conflicts by appending `-<number>`.

## Audience

Don't assume a fixed reader profile. Decide the depth of explanation from these rules, in order:

1. **The `audience:` arg, if the invoker passed one** (e.g. `audience: backend engineer new to both`). An explicit arg wins outright.
2. **Otherwise, your memory of the user's skillset** — the languages they're fluent in, _and the specific idioms you've already explained to them in past tours and recaps._ Explain a construct the first time it appears for this user; once memory records it as known, reference it plainly and don't explain it again. This is what stops the same idioms (closures, generics, error-handling idioms, …) being explained run after run as the user's fluency grows — step 6 keeps the record current.
3. **If you have neither an arg nor a relevant memory, treat every language as unfamiliar:** explain any genuinely non-obvious construct whatever its language, and state in the preamble that the audience was unspecified.

## Steps

1. **(Large scopes) Work in isolation.** For a whole-project tour, update the trunk and create a worktree or branch under `.worktrees/`, then `cd` into it and confirm with `git branch --show-current` before writing. Skip for a small subsystem doc.
2. **Ground in the canonical sources first.** Read the domain/architecture anchors before writing a word, so the document inherits the project's own vocabulary and structure: the domain-model doc, the contributor/agent guide, the relevant entrypoints, and the decision records (ADRs). Read any existing tour as a _style_ reference, not content to copy.
3. **Fan out, then synthesize.** A whole-project tour is too much for one pass — dispatch parallel subagents, each owning one slice (a server area, a client area, the decision-record set, specs, build tooling), each returning verbatim excerpts **with `file:line` labels** and the load-bearing facts. Then write the document in one narrative voice; do not staple reports together.
4. **Write to the house rules** (below).
5. **Land it.** Commit as `docs:` with an explicit pathspec, using the `commit` skill, and a body that summarizes the document and names any code/doc divergences it flagged. Open a PR only when asked.
6. **Record what you explained** (only when the audience came from audience rule 2 or 3 — not from an `audience:` arg, which frames a one-off reader and says nothing about the user). Append every idiom you explained for the first time this run to the user's skillset memory, so the next tour or recap references it plainly instead of explaining it again.

## Writing rules

- **Terminology first.** Open with the ubiquitous language so later sections use precise terms. Pull them from the domain doc; do not invent synonyms.
- **Broad strokes, then rabbit holes.** Each part states the shape, then dives. Err toward over-specificity — the job is to be exhaustive.
- **Show the code.** When discussing code, paste it verbatim in a fenced block labelled `path:line`. Excerpts must match source exactly; quote, then explain around the block. Highlight the interesting hunks, don't dump whole files.
- **Diagram the processes.** For any non-trivial flow, include a Mermaid diagram (sequence, state, or ER as fits). Diagrams illustrate; prose carries detail.
- **Verify divergences empirically.** Where code and docs/ADRs disagree, confirm the truth at the keyboard before writing it, and flag the divergence in the document rather than silently picking one.
- **Respect prose style.** Honor the project's prose guide, or the `write-prose` skill's fallback when it has none — inclusive language, no over-hyphenation, no editor-specific section banners.

## Notes

- This differs from the `recap` skill: that narrates a range of commits; this builds a from-the-source guide to how the system works.
