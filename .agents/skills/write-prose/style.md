# Writing prose

> Conventions for prose — documentation, code comments, commit messages — independent of any one language.

## Code comments

A code comment says what the code does and, where it isn't obvious from the names and types, the compressed _why_ — and it **never references a document outside the code**. No `docs/adrs/…`, no `docs/specs/…`, no bare `ADR 0007`. The citation is noise on every read and rots whenever the document is renumbered or superseded; a reader who wants the full reasoning behind a design opens `docs/adrs/`, which exists for exactly that. References are one-directional: an ADR or spec may point at code, but code never points back at a doc.

Succinctness is a strong default, not a hard line cap, and the bar is high: trim every comment to its floor and apply the litmus strictly — _would the next editor break something they can't see from the code?_ If no, the comment should not exist. **Private functions almost never pass it**: a private definition is named for its reader, so default to no comment and let the name and types carry it; reach for one line only when a genuine gotcha — an ordering dependency, a deliberate absence, a footgun — can't be shown in code. When a comment does survive it states the _what_ in one line, not the _why_ in a paragraph. A comment that merely restates the name, or that just helps you skim, is worse than none. A backticked reference to another code symbol is allowed but stays rare and plain, never a "mirrors X because Y" aside. If a comment is there to translate a name — a unit, a default, what a function returns — the name is wrong; rename the definition rather than annotate it. And when an ordering dependency earns its one line, write the dependency and its reason and nothing else — `Start before X, so …` — not a gloss on the surrounding values.

**Wrap comment prose at 80 columns.** The code formatter might not do it for you. Eighty keeps comments readable in side-by-side diffs.

## Markdown

**Never hard-wrap Markdown prose.** One line per paragraph, per list item and per table row; the renderer does the wrapping, and long lines are correct. This holds for every Markdown file in the tree — ADRs, specs, tickets, research notes, this document. A deliberate single-purpose short line is a different thing and stays: the project's domain glossary (conventionally `CONTEXT.md`) puts `**Term**:` and `_Avoid_:` each on their own line because they're structure, not wrapped prose.

This is the one place the 80-column rule doesn't reach, and the difference is who does the wrapping. A comment and a commit body are read where nothing will reflow them — a side-by-side diff, `git log` in a terminal — so they carry their own breaks. Markdown is rendered, so a fixed column buys nothing and costs a diff that reflows an entire paragraph when a word changes in its first sentence.

Prose drafted by an agent arrives hard-wrapped by default, so say this explicitly when delegating any writing task. It has leaked in that way before, and the correction is a whole-file diff.

## Hyphenation

Don't hyphenate established multi-word terms, or phrases sitting in noun position. Reserve hyphens for genuine compound modifiers, and even then only where omitting one would be ambiguous. When in doubt, match the form already used in the tree.

## Settled terms

Word choice splits across two documents, and it splits by kind.

The project's domain glossary (conventionally `CONTEXT.md`) is the authority on its domain nouns and each entry's `_Avoid_:` list governs prose wherever those nouns appear.

This section and the table below rule on everything else: spelling, capitalization, and words to avoid for reasons that have nothing to do with the project domain. Where a term is a domain noun the glossary wins, and a ruling here never overrides an entry there.

- **Unix domain socket** — not "Unix-domain-socket". The initialism **UDS** is fine.
- **coding agent** — not "coding-agent".
- **Git** — not "git" in prose. It's a proper noun ("a Git repository", "Git resolves the ref"), so lowercase only where it's literal code or part of an identifier: the `git` command and its invocations (`git worktree add`), filenames like `.gitignore`/`.git`, and coined lowercase terms like "gitignored".
- **ID** — not `id` or `Id` in prose, and **IDs** in the plural. It's an initialism, and the technical style guides rule the same way: [Google](https://developers.google.com/style/word-list) says "Not `Id` or `id`, except in string literals or enums", and Microsoft and Apple both spell it `ID`. The carve-out is Git's — lowercase only where it's literal code or part of an identifier: JSON-RPC's `id` member, a field spelled `sessionId`, a path component like `sessions/<session-id>/`. So "the server mints the session ID", but "a message with no `id`".

## Inclusive language

Reference the following guide to ensure non-inclusive language does not find its way into prose.

| Word / phrase                    | Category                 | Why to avoid                                                                                                                                           | Better alternatives                                                           |
| -------------------------------- | ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------- |
| blacklist / whitelist            | Racist / colonial        | Encodes white = good, black = bad. Reinforces racial bias in technical systems regardless of intent.                                                   | allowlist / denylist, blocklist / permitlist                                  |
| master / slave                   | Racist / colonial        | Directly evokes chattel slavery. "Slave" is never not associated with slavery. Originated from systems mirroring racial and colonial power structures. | primary / replica, leader / follower, controller / worker, parent / child     |
| master (branch)                  | Racist / colonial        | Carries connotations of dominance, ownership, and control. Impossible to fully decouple from master/slave framing in computing contexts.               | main, trunk, primary, source                                                  |
| grandfathered                    | Racist / colonial        | Originates from post-Civil War "grandfather clauses" used to suppress Black voting rights by exempting white voters from new registration rules.       | legacy, exempt, carried over, established, rollover                           |
| redline / redlining              | Racist / colonial        | Refers to the systematic denial of services (banking, insurance, housing) to non-white neighborhoods. Racist by origin.                                | review, assess, evaluate, track changes, annotate                             |
| black hat / white hat / gray hat | Racist / colonial        | Assumes white = good, black = bad. Carries racial meaning even when borrowed from Western film tropes.                                                 | ethical / unethical hacker, security researcher / malicious actor             |
| first-class citizen              | Racist / colonial        | Derives meaning from the existence of "second-class citizens" — a term tied to systematic racial and colonial discrimination.                          | core feature, built-in, top-level, native support                             |
| crazy / insane (intensity)       | Ableist                  | Appropriates mental health conditions to describe intensity or disorder. Reinforces the idea that mental illness is bad or inferior.                   | overwhelming, hectic, intense, baffling, chaotic, unbelievable                |
| cripple / crippled               | Ableist                  | Uses physical disability as a metaphor for dysfunction. Dehumanizes people with mobility impairments.                                                  | frozen, halted, blocked, degraded, hobbled, limited                           |
| lame                             | Ableist                  | Historically referred to people with mobility impairments; repurposed to mean uninteresting, equating disability with being lesser.                    | dull, uninspiring, disappointing, weak, poor                                  |
| dumb                             | Ableist                  | Originally meant a person who cannot speak. Both meanings denigrate disability.                                                                        | simple, basic, plain, unintelligent                                           |
| dummy                            | Ableist                  | One definition is a person who cannot speak; also used to mean ignorant. Common in code but has offensive origins.                                     | placeholder, mock, stub, no-op, fake, unused                                  |
| sanity check                     | Ableist                  | Frames mental illness as error and neurotypicality as correctness. Ties a medical condition to the concept of software quality.                        | quick check, confidence check, smoke test, validity check                     |
| sane / insane (in code)          | Ableist                  | Using "sane" to mean correct implies that being neurodivergent or mentally ill means being broken or wrong.                                            | valid, sound, sensible, reasonable, correct, well-formed                      |
| blind (metaphorical)             | Ableist                  | Using "blind" to mean unaware perpetuates negative stereotypes about visual impairment, conflating disability with ignorance.                          | unaware, uninformed, oblivious, inattentive                                   |
| colorblind (non-medical)         | Ableist                  | Medical appropriation of a real condition to describe racial indifference. Also a microaggression that erases lived racial experiences.                | race-neutral (contested), equitable, anti-racist                              |
| tone deaf                        | Ableist                  | Medical appropriation of an auditory condition used to describe social insensitivity. Equates disability with poor judgment.                           | insensitive, oblivious, out of touch, inconsiderate                           |
| OCD (casual use)                 | Ableist                  | Trivializes obsessive-compulsive disorder — a serious mental health condition — by using it as shorthand for neatness or thoroughness.                 | detail-oriented, thorough, meticulous, precise                                |
| derpy                            | Ableist                  | Dehumanizes people with intellectual/developmental disabilities and physical differences. Contributes to stigmatization.                               | clumsy, silly, goofy, foolish                                                 |
| retarded / r-word                | Ableist                  | Former clinical term now used as a slur. Removed from the DSM-5 (2013). Deeply harmful to I/DD communities.                                            | delayed, degraded, broken (in specific technical contexts; never of a person) |
| junkie                           | Ableist                  | Dehumanizes people with substance use disorder, a recognized medical condition. Casual use dismisses the seriousness of addiction.                     | enthusiast, fan, devotee, aficionado                                          |
| man-in-the-middle                | Gendered                 | Unnecessarily gendered term for a network attack concept that requires no gender reference.                                                            | on-path attacker, interceptor, person-in-the-middle (PITM)                    |
| man-hours                        | Gendered                 | Assumes a male default for labor. Excludes and erases workers of other genders.                                                                        | person-hours, labor-hours, work-hours, staff-hours                            |
| guys (mixed group)               | Gendered                 | Defaults to male when addressing mixed or unknown-gender groups. Subtly excludes non-male participants.                                                | folks, everyone, team, all, y'all                                             |
| he / she (generic)               | Gendered                 | Assumes binary gender when referring to users, readers, or developers. Excludes non-binary people.                                                     | they / them, the developer, the user, the reader                              |
| tribal knowledge                 | Culturally appropriative | Appropriates "tribal" — a term tied to indigenous peoples — and uses it reductively to mean informal or undocumented knowledge.                        | institutional knowledge, undocumented knowledge, tacit knowledge              |
| powwow (casual use)              | Culturally appropriative | A powwow is a sacred ceremony in many Indigenous North American cultures. Casual use for "meeting" is cultural appropriation.                          | meeting, huddle, sync, discussion, check-in                                   |
| spirit animal                    | Culturally appropriative | Appropriates a spiritual concept from Indigenous cultures to mean "something I identify with." Trivializes sacred belief systems.                      | personal icon, mascot, inspiration, something I identify with                 |
| guru                             | Culturally appropriative | A title of deep spiritual significance in Hindu and Sikh traditions. Casual use as "expert" strips its sacred meaning.                                 | expert, specialist, authority, pro, veteran                                   |

_Sources: [selfdefined.app](https://www.selfdefined.app), [Khronos Inclusive Language](https://www.khronos.org/about/inclusive-language), [Google Developer Style Guide](https://developers.google.com/style/inclusive-documentation)_

### Exceptions

Carve-outs are rare and must be justified.

- **Pseudoterminal terminology:**: POSIX names a PTY's two ends the **master** and the **slave**, and so do `openpty`, `ptsname`, and `pty(7)` . Prose describing that interface uses the interface's own words, because a renamed file descriptor describes an API that does not exist and costs the reader the term they would search for. The carve-out is narrow: it covers the two ends of a PTY and nothing else. Every other sense the table rules on is unaffected, including a branch named `master` and any role we are the ones naming — where we choose the word, the table's alternatives apply and this exception does not.
