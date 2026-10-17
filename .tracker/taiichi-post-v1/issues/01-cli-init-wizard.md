# CLI init wizard

Status: needs-triage

An interactive `taiichi init` that asks for the values it can't work out (the Forge's URL and repo, ready labels, position labels, comment switches) instead of taking them as flags. In v1, `taiichi init` writes `taiichi.toml` from the `origin` remote and flags, with every default spelled out; see [`taiichi init` and configuration](../../taiichi-v1/issues/09-init-and-configuration.md).
