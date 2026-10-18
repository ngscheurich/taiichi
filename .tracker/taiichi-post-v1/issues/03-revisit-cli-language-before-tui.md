# Revisit the CLI language before the TUI

Status: needs-triage

Before any work on the TUI starts, decide again whether the CLI stays in Go with Charm or moves to TypeScript on Bun, with Bombshell's libraries and OpenTUI. v1 chose Go; see [CLI language](../../taiichi-v1/issues/11-cli-language.md).

The checkpoint exists because switching is cheap until the TUI exists. The CLI is a thin client of the server's HTTP API, so a switch rewrites only its UI code and an 11-method API client. The TUI (Bubble Tea's model against OpenTUI's React components) and the wizard (Huh against a form layer of our own) don't carry over, so after them a switch costs much more, in either direction.

What to weigh then:
- **The harness:** whether it exists yet. It is a separate TypeScript process, so once it exists a TypeScript CLI would make the repo two languages instead of three and could share its API client.
- **Huh:** whether Huh has resumed releases and merged its v2 fixes (huh#831, huh#834).
- **The TypeScript stack:** whether Bombshell has shipped Clack theming and its TUI layer, whether Router's help has matured, and whether OpenTUI has reached 1.0 or gained a form library.
- **The prototypes:** both prototypes and their notes are on branch `prototype/cli-stacks`, in `cli/PROTOTYPES.md` and `cli/prototype-{go,bun}/NOTES.md`.
