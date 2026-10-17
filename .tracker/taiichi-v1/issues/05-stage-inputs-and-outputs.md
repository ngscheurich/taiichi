# Stage Inputs and Stage Outputs

Type: grilling
Status: open

## Question

What exactly is in each Stage's Stage Input and Stage Output?

- The required front-matter fields of the Blueprint, Build Notes, Evaluation (including the Verdict), and Manifest, and what goes in the Markdown body of each.
- What each Stage Input is made of: the copied Ticket for Plan; the Blueprint for Build; for a Build retry, the failed Evaluation; for rework after a Refused Shipment, the reviewer's feedback; the Build Notes for Evaluate; and what Publish receives.
- Whether a Stage Input is a single document or a bundle of earlier Stage Outputs, and how a later Attempt refers to earlier ones.

Format is settled as Markdown with YAML front matter (see [Chart the v1 scope](01-chart-the-v1-scope.md)).
