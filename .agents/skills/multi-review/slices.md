# Vertical slices

A _vertical slice_ is one feature traced through every layer it touches, from its entry point down to the edge where it meets the outside world (disk, network, a database, an external process). Reviewing a slice end to end — rather than a single file or a flat diff — is what surfaces the bugs that live _between_ layers: a value decoded wrong at a boundary, an error swallowed on the way back up, a contract two layers disagree on.

## Identify a slice by its stable identity

Name a slice by the domain concept and the entry point that drives it, not by source paths — paths drift, the concept and its surface do not. "Create an order" or "user login" is a slice; `src/foo/bar.ts:42` is not.

## Trace the file set at review time

Locate the actual files when you review, by following the feature down the stack:

1. **Start at the surface** — the command, route, handler, or UI action the user invokes.
2. **Follow each hop inward** — from the surface to the layer it calls, across any wire or process boundary, down to the next layer, and so on.
3. **Stop at the edge** — where the slice finally touches disk, the network, a database, or an external process.

The slice's file set is whatever that trace covers. Note the boundaries specially: any point where data crosses a wire/protocol seam or is decoded and validated is where cross-layer bugs hide, so include those surfaces in the file set even when the diff didn't touch them.

## Sharpening a diff or brief into a slice

A diff or a brief usually names only part of a slice — the files that changed, or the feature in the abstract. Widen it: from each changed file or named feature, trace outward to the surface that drives it and inward to the edge it reaches, so the reviewers see the whole vertical, not just the touched lines.

## Highest-yield targets

When choosing what to review most carefully, favor the slices that cross the most layers and carry the load-bearing logic — provenance, trust/security boundaries, state transitions, anything at an isolation or process boundary. Those exercise the rubric's correctness, design-fit, and safety dimensions hardest.
