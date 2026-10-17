# `taiichi init` and configuration

Type: grilling
Status: open
Blocked by: 07, 08

## Question

What does configuration look like, per repo and per server?

- The committed repo config file: its name and format (TOML was suggested but not chosen; YAML is already used for Stage data front matter, and JSON is built into Elixir), and its fields (Tracker, Forge, their options, and the Tracker's per-project mapping).
- What `taiichi init` does step by step, and how a repo is registered with the server and identified on later commands.
- The server's own config: data folder location, socket path, Forgejo URL and token.
