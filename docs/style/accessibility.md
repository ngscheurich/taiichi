# Accessibility in Taiichi

> How everything Taiichi says to people stays usable for everyone who reads it, whichever language produced it. For code style see [`go.md`](go.md) and [`elixir.md`](elixir.md); for prose see [`prose.md`](prose.md).

Most of what Taiichi says to people comes out of the `taiichi` CLI: tables, the Ticket block in `show`, rendered Stage Inputs, prompts and spinners. The server's error messages reach people too, printed by the CLI as they are, and so do the comments Taiichi posts on Forgejo. This guide treats accessibility as part of designing each of those, not a pass at the end. One rule underlies the rest: **no piece of meaning travels on only one channel a reader might not perceive**, whether colour, dimness, position or motion. Two groups of readers make that concrete, and each is let down in a different way:

- **Low vision or colour vision deficiency:** sighted, but unable to rely on colour or on fine contrast. Colour vision deficiency affects about 1 in 12 men ([National Eye Institute](https://www.nei.nih.gov/learn-about-eye-health/eye-conditions-and-diseases/color-blindness)).
- **Screen reader and braille display users:** they read the terminal's text in order, line by line, and perceive neither colour nor layout.

## Colour and dimming add to the words; they never replace them

- **Every styled state also has a word or symbol that carries its meaning alone.** Colour only makes it quicker to spot. `show` prints `Shipment closed`, not a red `Shipment`; a Verdict reads `pass` or `fail`, never just green or red. A glyph such as `●` sits beside a word (`● reachable`), never in its place.
- **For a screen reader, colour is not just hard to tell apart; it isn't there at all.** The terminal consumes the escape codes before the reader sees the text. Meaning carried only by colour fails both groups at once.
- **Make red against green never the only difference** between two states. Use different words, symbols or positions as well.
- **Use the 16 named ANSI colours** (`lipgloss.Color("1")` to `"15"`). The user's terminal theme decides what they look like, so they suit light and dark backgrounds without any work from us. Choose specific shades only when you must, and then pick between a light and a dark variant with `lipgloss.LightDark`, from one background check per run ([Lip Gloss v2](https://pkg.go.dev/charm.land/lipgloss/v2#LightDark)). Never assume the background is light or dark.
- **Faint text lowers contrast against a background you don't control**, and terminals handle it unevenly: some draw it dimmer, some ignore it. Use it only on text that is already secondary, such as a timestamp beside the thing it dates. Faintness itself must never be what tells the reader something is secondary.

## Output adapts to where it's going, and needs a lever to go plain

Output already changes with its destination: styled on a terminal, plain layouts and raw Markdown on a pipe ([`go.md`](go.md), section 14). A `--json` flag for programs is planned soon after v1. All of that is decided in one place, the CLI's rendering package, so every command degrades the same way.

- **Colour detection comes from colorprofile**, under Lip Gloss's writers. It honours `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE` and `TERM=dumb` ([colorprofile](https://pkg.go.dev/github.com/charmbracelet/colorprofile#Detect)). Don't reimplement any of them. It doesn't read `FORCE_COLOR`, a Node.js convention; `CLICOLOR_FORCE` is the setting that forces colour.
- **That detection decides colour only.** Borders, box-drawing and tree glyphs are content, and the rendering package decides on them separately, from whether stdout is a terminal. So `NO_COLOR` at a terminal still draws a table's borders, and a screen reader user at that terminal still hears them.
- **Treat `TERM=dumb` like a pipe for layout as well as colour.** colorprofile already strips all styling for it, and Huh already switches its prompts to accessible mode for it (below). Drawing borders in that case would be the one part left behind.
- **Taiichi has no explicit "plain" lever yet.** No portable signal tells a program a screen reader is attached, so a flag or setting the reader's user can turn on is the only dependable way to drop borders, in-place redraws and rendered Markdown at a terminal. Whether to add one, and what to call it, is an open question (below). Treat it as an accessibility feature when it's decided, not only a convenience for scripts.

## Screen readers and braille displays

- **Output is read top to bottom, left to right.** Borders, box-drawing and space-aligned columns are read out as noise or lose their structure. Keep the meaning in words: a line such as `Ticket #42 · closed · checked 12:04` survives being read aloud; a table cell whose meaning depends on its column header may not.
- **The plain path, and later `--json`, are main routes for assistive technology**, not just for scripts. Keep them complete: anything a styled layout shows, the plain one shows too.
- **Nothing redraws in place off a terminal**, or when plain output is asked for. A spinner that rewrites its line with `\r` and cursor moves floods a screen reader with repeats. Huh's spinner has a static mode for this (`WithAccessible(true)`), which prints its title once and no animation ([`huh/spinner`](https://pkg.go.dev/charm.land/huh/v2/spinner)).
- **Prompts have an accessible mode too.** A Huh form with `WithAccessible(true)` asks with plain numbered or `[y/N]` prompts instead of redrawing a form, and Huh turns this on by itself when `TERM=dumb` ([Huh](https://pkg.go.dev/charm.land/huh/v2#Form.WithAccessible)). Every prompt the CLI shows also has a flag that answers it, such as `--yes`, so nobody has to use a prompt at all.
- **Markdown for a screen reader is often best raw.** Glamour's rendering adds margins, bullets and heading styles a reader can't use. When plain output is wanted, print the Markdown as written, or render it with Glamour's `ascii` style, which uses only ASCII characters ([Glamour styles](https://pkg.go.dev/charm.land/glamour/v2/styles)).

## Glyphs, motion and timing

- **No emoji or uncommon Unicode carries meaning on its own.** Pair any glyph with text, have an ASCII fallback, and assume neither that the font has it nor that a screen reader says it sensibly.
- **Animation can be turned off**, along with the rest of the plain output. Motion distracts, can cause discomfort for readers sensitive to it, and floods screen readers with redraws. Never emit flashing output; in a text interface it's easy never to need it.
- **Prompts never time out.** Don't assume a reader is quick, sighted, or reading at all. Huh's accessible mode refuses timeouts anyway (`ErrTimeoutUnsupported`).

## What the server writes for people

The server's error messages and the comments Taiichi posts on Forgejo are read by people too, the first through the CLI, the second in a browser.

- **An error message says what happened and what to do, in words**, with no reliance on how it will be styled. The CLI prints it as it is, so whatever the message assumes about formatting, every reader inherits.
- **Forgejo comments carry their meaning in text**: "Shipped again: `abc123..def456`", not an emoji or a coloured badge. Use Markdown structure (headings, lists) for structure, so a screen reader in the browser can move through it.

## The post-v1 terminal UI

A full-screen Bubble Tea interface is planned after v1. When it's built:

- **Everything is reachable from the keyboard**, with no mouse-only path. Key bindings are listed in a help overlay and stay consistent between screens.
- **Focus is shown by more than colour**, moves in a sensible order, and quitting or backing out always works. On resize, the layout reflows rather than cutting off what matters.
- **The CLI stays the accessible way in.** A full-screen terminal program is close to unusable with a screen reader, so every operation in the terminal UI keeps a CLI equivalent, and a `--json` form once that exists.

## Where this is enforced

The CLI's rendering package is the single place output is styled, so the rule lives there: styles add colour to text that already carries the meaning, and commands write through the package, not through raw Lip Gloss, so the rule holds without anyone remembering it. Its tests render each styled state through a writer that isn't a terminal and check that the word carrying the meaning is still there.

## Open questions

1. **A plain lever.** Whether the CLI gets a `--plain` flag, an environment variable, or both, to drop borders, animation and rendered Markdown at a terminal and turn on Huh's accessible mode. An environment variable reaches every invocation without typing; a flag shows up in `--help`. Decide before the first table lands, so every command gets it from the rendering package at once.
