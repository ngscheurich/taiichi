# Review Rubric

The standard every reviewer holds the slice against, and the scale the judge scores reviews on. Read the cited project docs directly — this rubric points at the canon, it does not restate it. If a review dimension relies on a document (the domain glossary, a style guide) you were not handed, skip that rule — don't go searching.

## Dimensions

Score the slice on each. A strong review covers all ten, weighted toward where the slice actually lives.

Several dimensions below name a wire or process boundary. Where a slice has none — a library, a CLI, a compiler, a data transform — read those rules as their in-process equivalent: deserialization, parsing, and any point where a value crosses from an untrusted source into code that trusts it. The boundary is wherever data changes hands, not only where it crosses a socket.

1. **Correctness** — Does it do what it claims across the whole slice, including error paths, edge cases, and concurrency? Trace the data end to end. Decode failures, partial writes, crashes mid-operation, and data lost or garbled where it changes hands are correctness bugs, not nits.
2. **Design & domain fit** — Does it match the domain model and the relevant ADRs/specs? Right seams, right layer for each responsibility, no concept invented where one already exists. Flag drift from a documented decision, and flag code that contradicts a spec.
3. **Idiom & style** — Is it idiomatic per the project's language style guide(s)? Prefer standard idioms over reinvention, and parse or validate incoming data at the boundary rather than deep inside. Flag non-idiomatic constructs.
4. **Prose** — Do comments and strings follow the project's prose style guide?
5. **Tests** — Do tests exist and cover the behavior, including the failure paths? Are they at the right level and deterministic? Flag missing coverage on changed behavior and tests that assert nothing meaningful.
6. **Safety & boundaries** — Resource handling, filesystem assumptions, and untrusted input wherever it enters — a wire surface, a parsed file, a command-line argument, a caller outside the module. Flag unbaked assumptions.
7. **Dead code** — Code that nothing reaches or needs: unused functions, types, bindings, imports, and config; unreachable branches; commented-out blocks; parameters that are always passed the same value; behind-a-flag paths no longer wired up. Flag it for removal rather than preservation — the history keeps it.
8. **Naming** — Names that mislead or obscure. Flag an identifier whose name implies behavior it does not have (a `get_` that mutates, a `_count` that is a list), and one so obtuse the reader must read the body to learn what it holds. Names state the plain observable fact (per the project's prose style guide and its domain glossary); flag drift from the domain term and invented synonyms for an existing concept.
9. **Performance** — Likely, not theoretical, performance problems on the real path: O(n²) over a list that grows, work repeated inside a loop that hoists out, an unbounded buffer or read, a full scan where a lookup exists, syscalls or process spawns in a hot path. Flag the input that makes it bite; do not chase micro-optimizations with no plausible scale.
10. **Comments** — Hold every comment to the `sweep-comments` bar (the rules live in the project's prose style guide, or the `write-prose` skill's fallback when it has none): a comment exists only to protect the next editor from breaking something they _cannot see in the code_. Flag a comment that restates the body, restates the identifier's name, documents an external tool's behavior, cites a doc/ADR or recounts history/roadmap, or translates a name (the name is wrong — note it for the naming dimension). Genuine gotchas — an ordering dependency, a deliberate absence, a wire invariant a concurrent reader relies on — stay. A comment that earns its place but is missing the _why_ is a finding too.

## Severity scale

- **blocker** — ships a bug, data loss, or a security/safety hole; or violates a load-bearing documented decision. Must fix before merge.
- **major** — wrong in a real scenario, a meaningful design problem, or missing tests on changed behavior. Should fix before merge.
- **minor** — narrow correctness gap, idiom drift, dead code, a misleading name, or a maintainability problem.
- **nit** — style/prose/naming/comment polish. No functional impact.

## Finding format

Every finding is one entry:

```
### [severity] short title
- **where:** path/to/file:120-134
- **dimension:** correctness | design | idiom | prose | tests | safety | dead-code | naming | performance | comments
- **problem:** what is wrong, concretely — the specific input/path that breaks
  or the specific rule/decision it violates.
- **fix:** the concrete change you would make.
```

A reviewer's return value is: a one-paragraph overall read, then findings sorted by severity (blocker → nit). Cite `file:line` for every finding; a finding with no location or no concrete problem does not count.

## How the judge scores a review

Rank reviews on, in order:

1. **Accuracy** — findings are real and correctly diagnosed. Wrong or speculative findings are penalized harder than misses.
2. **Coverage** — real blockers and majors caught across the dimensions, not just a pile of nits.
3. **Prioritization** — severities are honest; the important thing is at the top.
4. **Actionability** — fixes are concrete and correct.

The winner is the review that is most accurate _and_ most complete — not the longest.
