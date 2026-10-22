## Agent skills

### Issue tracker

Issues live in Forgejo (`ngscheurich/taiichi` on the host), driven through `scripts/forgejo`; needs `FORGEJO_TOKEN`. See `docs/agents/issue-tracker.md`.

### Triage labels

Uses the five default triage roles as-is (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Style guides

Before writing or reviewing, read the guide for what you're touching:

- Elixir in `server/`: `docs/style/elixir.md`.
- Go in `cli/`: `docs/style/go.md`.
- Anything people read from Taiichi (CLI output, prompts, server error messages, Forgejo comments): `docs/style/accessibility.md`.
- Prose (docs, code comments, commit messages): `docs/style/prose.md`.

A guide's "Conventions" are house rules. Its "Open questions" are undecided: ask the user before settling one.
