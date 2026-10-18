# SQLite for the server's data

The server keeps Repos, Jobs, Attempts, Stage Inputs, and Stage Outputs in one SQLite database (`$XDG_DATA_HOME/taiichi/taiichi.db`) through Ecto, not as Markdown files with YAML front matter in a data folder, as first charted. Worktrees stay on disk. With files, every move needed a temporary folder and a rename, a cleanup of leftovers at startup, a rule to finish half-done moves, and an in-memory catalog to enforce rules across Jobs. A transaction and unique indexes do all of that, and SQLite syncs its own writes, including the folder sync Erlang can't do.

## Considered Options

- **Markdown files in a data folder** (the charted design): readable with `ls` and `cat`, and a Repo could be removed by deleting one folder, but all the crash handling had to be written by hand.
- **PostgreSQL:** a second server to install, start, and configure, and tests and agent sandboxes would need one running. Its extras (several writers at once, network access) answer no stated need: writes are short and run one at a time, and remote deployment is out of scope.

## Consequences

- Neither option makes it easier to recover from an operation that stops partway. Pickup, Ship, and Abandon also act on git and Forgejo, which no transaction can include. No transaction stays open across a git or Forgejo call.
- Stored data is no longer readable without the CLI or `sqlite3`. A Repo is removed with a `DELETE` that cascades, plus removing its worktrees and running `git worktree prune`.
