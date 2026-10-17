# List and remove Repos

Status: needs-triage

A `taiichi repos` command that lists every registered Repo (path, Forge location, active Job count, and a flag when the path no longer exists), plus a way to remove one. In v1 a Repo can only be removed by hand; see [`taiichi init` and configuration](../../taiichi-v1/issues/09-init-and-configuration.md).

Removal has to settle:
- **Active Jobs:** refuse until they're abandoned, or abandon them as part of removal.
- **Job history:** delete it, or keep it so a later `taiichi init` brings it back (which makes removal closer to hiding).
- **Branches:** a fresh Repo's Job numbers restart at 1, so its branch names can match kept ones. Options: delete Taiichi's branches (this breaks the Abandoned comment's "the branch is kept", and loses unshipped work for good), rename them (a push and a delete on the Forge for each), or keep the removed Repo's next Job number so names never collide.
- **The Tickets:** whether the position labels left on them, such as `taiichi/done`, stay.
