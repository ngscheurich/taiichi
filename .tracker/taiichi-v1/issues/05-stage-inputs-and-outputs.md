# Stage Inputs and Stage Outputs

Type: grilling
Status: resolved

## Question

What exactly is in each Stage's Input and Output?

- The required front-matter fields of the Blueprint, Build Notes, Evaluation (including the Verdict), and Manifest, and what goes in the Markdown body of each.
- What each Stage Input is made of: the copied Ticket for Plan; the Blueprint for Build; for a Build retry, the failed Evaluation; for rework after a Refused Shipment, the reviewer's feedback; the Build Notes for Evaluate; and what Publish receives.
- Whether a Stage Input is a single document or a bundle of earlier Stage Outputs, and how a later Attempt refers to earlier ones.

Format is settled as Markdown with YAML front matter (see [Chart the v1 scope](01-chart-the-v1-scope.md)).

## Answer

Settled with the user. Terms are defined in `GLOSSARY.md`.

**Terms**
- Publish is renamed **Ship**: both the Stage and the Forge operation (the Forge's operations are Ship and Report).
- New terms: **Ticket Snapshot** (the Ticket copy taken at Pickup) and **Feedback** (the reviewer's comments on a Refused Shipment).
- Attempts are numbered within their Stage: Build 1, Build 2.

**Format**
- Submitters write plain Markdown. Structured values travel as separate fields of the submission, not as front matter. The server writes front matter into every saved document, so stored Stage data is Markdown with front matter. This refines the charted "Markdown with YAML front matter": only the server writes the front matter.
- Required fields are only those the server acts on. The server stamps identity itself: Job, Stage, Attempt, submitted-at.

| Document | Field | Supplied by | Checked |
|---|---|---|---|
| All four Stage Outputs | body | submitter | must not be empty |
| All four Stage Outputs | Job, Stage, Attempt, submitted-at | server | n/a (stamped) |
| Build Notes | commit range | server | worktree must be clean |
| Evaluation | verdict | submitter | required, `pass` or `fail` |
| Evaluation | commit judged | server | head unmoved since the Build Notes it judges |
| Manifest | title | submitter | optional; non-empty if given |
| Manifest | Shipment URL, Forge id, shipped commit, feedback cursor | server (from the Forge) | n/a |
| Ticket Snapshot, Feedback | everything | server | n/a |

**Shape**
- A Stage Input is a bundle: a folder per Attempt holding a small input document whose front matter lists its parts. New material (Ticket Snapshot, Feedback) is stored as its own documents. Earlier Stage Outputs are referenced by Stage and Attempt, never copied. The exact paths belong to [Server internals](12-server-internals.md).
- A Stage Output is one document in v1, with no attachments. Attachments could be added later as extra files in the Attempt's folder without changing what's already stored.

**Each document**
- **Ticket Snapshot:** front matter holds the Ticket's id, URL, title, labels, and when the snapshot was taken. The body is the Ticket's body, word for word. Comments are not included.
- **Blueprint:** body only.
- **Build Notes:** submission is rejected if the worktree has uncommitted changes. The server stamps the Attempt's commit range: from the head when the Attempt began (for Build 1, the default-branch commit the branch started from) to the head at submission. Each Build Notes describes only its own Attempt's commits.
- **Empty commit range:** allowed. Such a Build Attempt still writes fresh notes saying why nothing changed, never a re-submission of earlier notes. Examples: an Evaluation that failed against a stale environment, or Feedback that needs an explanation rather than a change. A re-evaluate shortcut was rejected because it would change the fixed Pipeline.
- **Evaluation:** verdict plus body. Rejected if the branch's head moved since the Build Notes it judges.
- **Manifest:** the Ship submission is an optional title (defaults to the Ticket's title) and the Shipment description (the body). The server adds the Shipment's URL and Forge id, the shipped head commit, and the feedback cursor. A later Ship Attempt updates the same Shipment and may leave out the title to keep the existing one. How the cursor works on Forgejo is left to [Forgejo Forge](08-forgejo-forge.md).
- **Feedback:** written by the server from the Forge's Report. Front matter names the Shipment and the commit that was reviewed. The body has one section per item: author, location (file and line, if inline), and text.

**Stage Inputs**
- **Plan:** the Ticket Snapshot.
- **Build:** the Blueprint, plus the latest reason it came back (the failed Evaluation, or the Feedback). Not the full history, since the worktree holds the earlier work.
- **Evaluate:** the Ticket Snapshot, the Blueprint, and the Job's history after Plan, in order: each Build Notes alongside the failed Evaluation or Feedback that sent the Job back to Build. Per-Attempt Build Notes don't tell the whole story on their own.
- **Ship:** the same as Evaluate, ending with the passing Evaluation.

## Comments

**From [Server internals](12-server-internals.md):** the server writes no front matter: documents are SQLite rows, and the values that were front matter are a JSON `meta` column. A Stage Input is stored as `input_parts` rows pointing at documents, in order, instead of an input document listing paths. Feedback is stored with the Build Attempt it opens.

**From [A Shipment changing under a Job](16-shipment-changing-under-a-job.md):**
- A Build Attempt's range starts at the last commit the server pinned: the Job's `base_commit` for Build 1, the judged commit after a failing Verdict, the shipped commit after a Refusal. This replaces "the head when the Attempt began", which nothing recorded.
- The range holds every commit that came into the branch during the Attempt, whoever wrote it, except commits reachable from the local `origin/<default>`. Build Notes also records the base move (`base: main@1111 → main@2222`) and the foreign commits (sha and author): those reachable from the local `origin/<job branch>` and not from anything Taiichi pushed.
- Ship is refused with `409 head_moved` when the head isn't the commit the passing Evaluation judged, unless submitted with `--unjudged`. The Manifest records the judged commit next to the shipped one.
