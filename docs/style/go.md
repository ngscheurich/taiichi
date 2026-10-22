# Writing Go in Taiichi

> A guide to writing idiomatic Go for Taiichi's CLI, drawn from the Go team's own guidance ([Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), [Go Doc Comments](https://go.dev/doc/comment), [Organizing a Go module](https://go.dev/doc/modules/layout), the Go blog and the release notes through Go 1.27), the two most widely followed company guides ([Google's Go Style Guide](https://google.github.io/styleguide/go/) and [Uber's Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)), Dave Cheney's [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html), the [Command Line Interface Guidelines](https://clig.dev/), and the libraries the CLI is built on: [Cobra](https://github.com/spf13/cobra), [Fang](https://github.com/charmbracelet/fang), [Huh](https://github.com/charmbracelet/huh), [Lip Gloss](https://github.com/charmbracelet/lipgloss) and [Glamour](https://github.com/charmbracelet/glamour).

Taiichi has two programs. The server, written in Elixir, holds every rule and is the only program that writes Job data. The CLI, `taiichi`, is written in Go and holds no rules of its own. Each command works out which Repo and Job the user means (from Git, and from the server's `locate` method when the Job number is left out), posts a JSON request to the server over HTTP on a Unix domain socket, and prints the answer for people. It also opens the user's editor to write Stage Outputs, and always sends the text itself, never a file path. When the server isn't there, it says so plainly. When the server answers with an error, the CLI prints the server's `message` as it is and branches only on the error's `code`. Those three duties (talking to the server, running `git`, and printing well on terminals and pipes) are where nearly all of the CLI's design pressure falls, and this guide spends most of its length on them.

How to read this guide: sections 1 to 9 are general Go and apply to every file. Sections 10 to 14 cover the CLI's own layers: the server, JSON, Git and the editor, commands, and output for people. Section 15 is testing, 16 logging and 17 comments. Section 18 lists the conventions this project adopts, and section 19 the questions it hasn't settled. A rule marked **provisional** is a recommendation nobody has signed off on yet; each one appears again in section 19. When in doubt, prefer what the standard library does over inventing something new.

---

## 1. The module and its layout

The CLI is one Go module in `cli/`, beside the server's Mix app in `server/`. Keeping both in one repository lets one commit change an API method and the CLI code that calls it. The Go toolchain is pinned in the root `mise.toml` (`go = "1.27.3"`), so `mise install` gives everyone the same version.

### The module path

A module path names the module and prefixes every import path in it. The Go team's advice is to use the repository's location if the module might ever be fetched by others, and otherwise a name you control, avoiding generic words ([Naming a module](https://go.dev/doc/modules/managing-dependencies#naming_module)). A module can live in a subdirectory of its repository, in which case the path gains that directory. Taiichi's repository is `github.com/ngscheurich/taiichi`, so the CLI's module path is **provisionally** `github.com/ngscheurich/taiichi/cli`.

```
// cli/go.mod
module github.com/ngscheurich/taiichi/cli

go 1.27
```

`go mod init` now writes a `go` line one minor version behind the toolchain ([Go 1.26](https://go.dev/doc/go1.26#go-command)). Raise it to `1.27` by hand: the `go` line turns on language features per module, such as per-iteration loop variables ([Go 1.22](https://go.dev/blog/loopvar-preview)), and it decides which `go vet` checks run.

### Directories

The Go team's layout page recommends keeping a server's or tool's packages in `internal/`, and putting commands under `cmd/` ([Organizing a Go module](https://go.dev/doc/modules/layout#server-project)). A directory named `internal` is special to the `go` tool: only code rooted at its parent can import it, so nothing outside the module can come to depend on it. The layout page doesn't mention a `pkg/` directory, and the CLI has no code meant for other modules, so it has none.

There's a second reason for `cmd/taiichi/`: `go build` names the binary after the last element of the main package's import path ([`go help build`](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)). A `main` package at the module root would build a binary called `cli`.

**Provisional** package layout:

```
cli/
  go.mod
  go.sum                  committed
  cmd/
    taiichi/
      main.go             wires things together and calls command.Execute; nothing else
  internal/
    command/              the Cobra command tree: one file per command
    api/                  the server's client, its request and result types, the socket path
    git/                  the git commands the CLI runs
    editor/               finding and running the editor, the scissors line
    render/               output for people: tables, key-value blocks, Markdown, prompts
  testdata/               fixtures and golden files; the go tool ignores this directory
```

Name packages for what they provide, and let them be fairly large: Go has no "one type, one file" convention ([Google best practices](https://google.github.io/styleguide/go/best-practices#package-size)), and Dave Cheney recommends fewer, larger packages over many small ones ([Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html)). Split a package when it holds two ideas a reader would look for separately, not when a file gets long.

### Commands

Run these from `cli/`:

| What | Command |
|---|---|
| Build the binary | `go build -o bin/taiichi ./cmd/taiichi` |
| Run without building | `go run ./cmd/taiichi status` |
| Test | `go test ./...` |
| Test with the race detector | `go test -race ./...` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run` |
| Update code to current idioms | `go fix ./...` |
| Check for known vulnerabilities | `go tool govulncheck ./...` |
| Tidy dependencies | `go mod tidy` |

## 2. Formatting and tooling

**`gofmt` is not optional.** Every Go file is formatted by `gofmt`, which has no settings ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#gofmt)). `goimports` does the same and also adds, removes and groups imports; set your editor to run it on save. No line length limit exists: break a line where the meaning breaks, not at a column ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#line-length)).

**`go vet`** catches mistakes the compiler allows: `Printf` verbs that don't match their arguments, copied locks, malformed test names, misplaced `WaitGroup.Add` calls and more. `go test` runs a subset of it automatically. Run the full set before every commit.

**`golangci-lint`** runs many linters at once and is the runner Uber recommends ([Uber: Linting](https://github.com/uber-go/guide/blob/master/style.md#linting)). Its v2 configuration starts from a `standard` set of five linters: `errcheck`, `govet`, `ineffassign`, `staticcheck` and `unused` ([reference configuration](https://golangci-lint.run/docs/configuration/file/)). Uber adds `revive` and `goimports` to that base. Several of `staticcheck`'s style checks are off by default, including ST1003, which flags `Id` and `Url` where Go wants `ID` and `URL` ([staticcheck checks](https://staticcheck.dev/docs/checks/)). `go vet` does not catch those, so turn ST1003 on. Install `golangci-lint` as a pinned binary through `mise`, not with `go install` or a `tool` line in `go.mod`: its maintainers say those builds "aren't guaranteed to work" ([install docs](https://golangci-lint.run/docs/welcome/install/local/)).

A **provisional** starting configuration, `cli/.golangci.yml`:

```yaml
version: "2"
linters:
  default: standard
  enable:
    - bodyclose       # an HTTP response body that's never closed
    - errorlint       # errors compared with == or type switches instead of errors.Is/As
    - revive          # common style mistakes
    - usestdlibvars   # http.MethodPost, not "POST"
  settings:
    staticcheck:
      checks: ["all", "-ST1000"]   # all checks, minus "every package needs a doc comment"
formatters:
  enable:
    - goimports
```

**`go fix`** was rewritten in Go 1.26 to carry the **modernizers**, fixers that move code to current language features and APIs without changing what it does: `interface{}` becomes `any`, `x := x` loop copies disappear, `errors.As` with a temporary variable becomes `errors.AsType`, a counted `for` loop becomes `for range n`, and about twenty more ([Go 1.26](https://go.dev/doc/go1.26#go-command), [modernize](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/modernize)). Run it after upgrading the toolchain, and review the diff like any other change.

**`govulncheck`** reports known vulnerabilities in the module's dependencies, and separates the ones your code can reach from the ones it merely imports ([tutorial](https://go.dev/doc/tutorial/govulncheck)). Track it as a module tool, a feature since Go 1.24 that replaced the old `tools.go` file ([Go 1.24](https://go.dev/doc/go1.24#tools)): `go get -tool golang.org/x/vuln/cmd/govulncheck`, then `go tool govulncheck ./...`.

**The race detector** (`go test -race`) finds two goroutines touching the same memory without synchronisation. The CLI has little concurrency, but the HTTP client and spinners do run goroutines, and the check is cheap at test time.

## 3. Packages and imports

Every file in a directory declares the same package, and the directory is the package. Package names are short, lowercase, one word, with no underscores or mixedCaps: `api`, `editor`, `render` ([Effective Go](https://go.dev/doc/effective_go#package-names)).

```go
// Package api is the CLI's client for the Taiichi server's HTTP API.
package api

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"net/http"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"github.com/ngscheurich/taiichi/cli/internal/git"
)
```

- **Group imports** into standard library, third-party, and this module, separated by blank lines. `goimports` does it for you ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#imports)).
- **Don't rename imports** except to resolve a collision, and then rename the more local one ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#imports)). A version suffix doesn't count: `charm.land/lipgloss/v2` is already package `lipgloss`.
- **No dot imports** (`import . "fmt"`). They hide where a name comes from ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#import-dot)).
- **Blank imports** (`import _ "embed"`) belong only in `main` or in tests ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#import-blank)).
- **No grab-bag packages.** `util`, `common`, `helpers`, `misc`, `types` and `api`-as-a-dumping-ground tell the reader nothing and grow without limit ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#package-names), [Google best practices](https://google.github.io/styleguide/go/best-practices#util-packages)). A helper belongs in the package that uses it, or in a package named for what it does. (`api` here is fine because it names one thing: the server's API.)
- **No `init` functions.** They run before `main`, in an order that depends on imports, and can't return errors ([Uber: Avoid `init()`](https://github.com/uber-go/guide/blob/master/style.md#avoid-init)). Build things explicitly in functions that `main` calls.

## 4. Naming

The compiler enforces one rule: a name that starts with an uppercase letter is exported from its package, and anything else is private to it. Everything below is convention, but every Go reader expects it.

- **MixedCaps, never underscores**: `stageOutput`, `ParseJobNumber`, `maxSlugLength` ([Effective Go](https://go.dev/doc/effective_go#mixed-caps)). Test, benchmark and example function names are the exception (`TestSubmit_jobMoved`) ([Google decisions](https://google.github.io/styleguide/go/decisions#underscores)).
- **Initialisms keep one case**: `ID`, `URL`, `HTTP`, `JSON`, `TTY`. So `jobID`, `TicketURL`, `newHTTPClient`, never `jobId` or `TicketUrl` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#initialisms)).
- **Don't stutter.** A caller writes the package name, so `api.Client`, not `api.APIClient`; `api.New`, not `api.NewClient` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#package-names)). The standard library does this throughout: `http.Client`, `bytes.Buffer`.
- **Getters have no `Get` prefix.** A method returning the `branch` field is `Branch()`. Use a verb such as `Fetch` when the call does real work, so readers don't mistake it for a field read ([Effective Go](https://go.dev/doc/effective_go#Getters), [Google decisions](https://google.github.io/styleguide/go/decisions#getters)).
- **Name length follows scope.** One letter for a receiver or a three-line loop; a full word for anything a reader meets far from its declaration ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#variable-names), [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html)). Don't drop letters to save typing: `attempt`, not `atmpt` ([Google decisions](https://google.github.io/styleguide/go/decisions#variable-name-length)).
- **Don't put the type in the name**: `jobs`, not `jobSlice` or `jobList` ([Google decisions](https://google.github.io/styleguide/go/decisions#repetitive-with-type)).
- **Receivers** are one or two letters that abbreviate the type, the same on every method: `c *Client`, `j Job`. Never `self` or `this` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#receiver-names)).
- **Errors**: a sentinel error value starts with `Err` (`ErrNotARepository`); an error type ends in `Error` (`UnreachableError`) ([Uber: Error Naming](https://github.com/uber-go/guide/blob/master/style.md#error-naming)).
- **One-method interfaces** take the method name plus `-er`: `Reader`, `Locator` ([Effective Go](https://go.dev/doc/effective_go#interface-names)).
- **Boolean functions read as questions**: `IsTerminal`, `CanPrompt`, `HasShipment`. A doc comment for one says what it "reports whether" ([Go Doc Comments](https://go.dev/doc/comment#func)).
- **Domain words come from `GLOSSARY.md`**, spelled the way it spells them and never with a word from an entry's _Avoid_ list: `Ticket`, not `Issue`; `Attempt`, not `Run` or `Pass`; `Position`, not `State`. The glossary is the authority on these nouns in code as well as prose.

## 5. Functions, methods and receivers

```go
// Submit sends a Stage Output for the Attempt it was written for. If the Job
// has moved on since, the server refuses with the code job_moved.
func (c *Client) Submit(ctx context.Context, p SubmitParams) (*SubmitResult, error) {
	var out SubmitResult
	if err := c.Call(ctx, "submit", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

- **`context.Context` comes first**, named `ctx`, on anything that does I/O or might block. Pure functions, such as building a branch slug or cutting text at the scissors line, take none ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#contexts)).
- **`error` comes last**: `(T, error)`, never `(error, T)` ([Google decisions](https://google.github.io/styleguide/go/decisions#returning-errors)).
- **Handle the error first and return early**, so the normal path stays at the left margin ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#indent-error-flow)). No `else` after a `return`.
- **Named results** only when two results share a type and the names say which is which, or when a deferred function must change a result. **Naked `return`** only in functions a few lines long ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#named-result-parameters)).
- **Prefer synchronous functions.** A function that does its work and returns is easier to test and to reason about, and a caller who wants concurrency can add it ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#synchronous-functions)).
- **Avoid bare `bool` arguments** whose meaning a call site can't show. `client.Jobs(ctx, gitDir, true)` hides what `true` means; a parameter struct with an `All` field, or two functions, doesn't ([Uber: Avoid Naked Parameters](https://github.com/uber-go/guide/blob/master/style.md#avoid-naked-parameters)).
- **Many arguments become a struct.** Google prefers an options struct as the last parameter; functional options (`New(opts ...Option)`) suit an API where most callers configure nothing and settings keep growing ([Google best practices](https://google.github.io/styleguide/go/best-practices#function-argument-lists), [Uber: Functional Options](https://github.com/uber-go/guide/blob/master/style.md#functional-options)). The CLI has no public API and few such constructors, so reach for a plain struct first. Every API method already takes its parameters as a struct (sections 10 and 11).

### Pointer or value receiver

- **Pointer receiver** if the method changes the receiver, if the type holds something that mustn't be copied (a `sync.Mutex`, a `bytes.Buffer`), or if the type is large. **When in doubt, use a pointer** ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#receiver-type)).
- **Value receiver** for small values that behave like numbers or strings: `Position`, `Stage`, a decoded `Job` you only read.
- **Don't mix them on one type.** If any method needs a pointer, give all of them one.
- **Don't pass a pointer just to save copying a few bytes** ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#pass-values)). Pointers to strings, slices, maps and interfaces are almost never needed, with one exception: a `*string` field that must encode as JSON `null` (section 11).

## 6. Types, values and literals

**Make the zero value useful.** A `var buf bytes.Buffer` or a `var mu sync.Mutex` works with no constructor, and types you write should aim for the same ([Effective Go](https://go.dev/doc/effective_go#allocation_new), [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html)). Where a type needs a constructor, say so in its doc comment.

**Declare with `var` for a zero value and `:=` for anything else** ([Google best practices](https://google.github.io/styleguide/go/best-practices#vardecls)):

```go
var jobs []api.Job           // nil slice, ready to append to
out := new(bytes.Buffer)     // fine, but var out bytes.Buffer reads better
client := api.New(socket)
```

**Prefer nil slices.** `var labels []string` over `labels := []string{}`, and test for emptiness with `len(labels) == 0`, never `labels == nil` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#declaring-empty-slices), [Uber: nil is a valid slice](https://github.com/uber-go/guide/blob/master/style.md#nil-is-a-valid-slice)). How a nil slice encodes as JSON depends on the encoder; see section 11.

**Struct literals name their fields**: `api.SubmitParams{Job: 7, Stage: api.StageBuild}`, never `api.SubmitParams{7, "build", ...}`. A positional literal breaks silently when a field is added or reordered ([Uber: Initializing Structs](https://github.com/uber-go/guide/blob/master/style.md#use-field-names-to-initialize-structs)). Leave out fields that are zero unless naming them says something.

**Wire enums are string types with constants.** The API sends Positions, Stages, Verdicts and Reports as lowercase strings, so model each as its own string type. The type keeps a Position from being passed where a Stage is wanted, and the constants keep spelling in one place:

```go
// Position is where a Job stands: one of the four Stages, or one of the two
// ends.
type Position string

const (
	PositionPlan      Position = "plan"
	PositionBuild     Position = "build"
	PositionEvaluate  Position = "evaluate"
	PositionShip      Position = "ship"
	PositionDone      Position = "done"
	PositionAbandoned Position = "abandoned"
)
```

Uber's advice to start `iota` enums at one, so the zero value isn't a valid member by accident, applies to integer enums ([Uber: Start Enums at One](https://github.com/uber-go/guide/blob/master/style.md#start-enums-at-one)); the empty string plays that role here.

**Use `time.Time` and `time.Duration`** for instants and spans, never integers of seconds ([Uber: Use "time" to handle time](https://github.com/uber-go/guide/blob/master/style.md#use-time-to-handle-time)). The API's timestamps are RFC 3339, which `time.Time` decodes as it is.

**Embedding** a type in a struct promotes its methods. Use it to mix in a capability, not to model "is a kind of", and don't embed types in exported structs, where the embedded type's whole method set becomes part of yours ([Uber: Avoid Embedding Types in Public Structs](https://github.com/uber-go/guide/blob/master/style.md#avoid-embedding-types-in-public-structs)).

**Generics** fit when one function genuinely serves several types, such as the standard `slices` and `maps` packages. Don't write a generic helper for a single type, and reach for `slices.Contains`, `slices.SortFunc` and the like before writing your own ([Google decisions](https://google.github.io/styleguide/go/decisions#generics)).

## 7. Interfaces

Go interfaces are satisfied implicitly: any type with the right methods fits, with no `implements` declaration. That makes the conventions differ from Java or C#.

- **Define an interface where it's used, not where it's implemented** ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#interfaces)). If `command` needs only to look up a Job's number, `command` declares a one-method interface; `api` exports a concrete `*Client`.
- **Accept interfaces, return concrete types.** A function that writes output takes an `io.Writer`, so callers can pass a terminal, a pipe or a `bytes.Buffer`. A constructor returns `*Client`, so callers get its whole API ([Google decisions](https://google.github.io/styleguide/go/decisions#interfaces)).
- **Don't declare an interface before there's a second implementation**, and don't declare one only so a test can fake it. Test through the real type where you can ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#interfaces)). For the server, that means a real `*api.Client` talking to a test server, not a fake client (section 15).
- **Keep interfaces small.** The standard library's most used ones have one method.
- **Never take a pointer to an interface** ([Uber: Pointers to Interfaces](https://github.com/uber-go/guide/blob/master/style.md#pointers-to-interfaces)).

## 8. Errors

Go has no exceptions. A function that can fail returns an `error` as its last result, and the caller checks it.

```go
dir, err := git.CommonDir(ctx, cwd)
if err != nil {
	return fmt.Errorf("find the Repo: %w", err)
}
```

### 8.1 `if err != nil` stays

Several proposals tried to shorten this. In 2025 the Go team stopped pursuing any syntax change for error handling and began closing such proposals ([No syntactic support for error handling](https://go.dev/blog/error-syntax)). Write the check out; don't build helpers to hide it.

### 8.2 Add context, and choose `%w` or `%v`

When an error crosses a layer, add what the lower layer didn't know: which command, which Job, which file. Don't repeat what the error already says, and don't add "failed to" ([Google best practices](https://google.github.io/styleguide/go/best-practices#error-extra-info)).

- **`%w` wraps**: the new error keeps the old one inside it, where `errors.Is` and `errors.AsType` can find it. Wrapping makes the inner error part of your function's contract, so wrap only when callers should be able to inspect it ([Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors#whether-to-wrap)).
- **`%v` formats**: the text survives, the inner error doesn't. Use it when the cause is an implementation detail.
- **Put `%w` at the end** (`"read Stage Input: %w"`), so the text reads in chain order ([Google best practices](https://google.github.io/styleguide/go/best-practices#error-percentw)).

### 8.3 Error strings

Lowercase, no final punctuation, because they get wrapped into longer messages: `"find editor: %w"`, not `"Failed to find editor."` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#error-strings)). A message the CLI shows a person as a complete sentence is different, and is written as one (section 14.5). The server's messages are already sentences, and the CLI prints them untouched.

### 8.4 Sentinels and error types

- **A sentinel** is a package-level value for a failure the caller only needs to recognise: `var ErrNotARepository = errors.New("not inside a Git repository")`. Compare with `errors.Is`.
- **An error type** carries data the caller wants: the server's code and details, the socket path that didn't answer. Extract it with `errors.AsType`, new in Go 1.26, which returns the typed value and a boolean and needs no temporary variable ([`errors.AsType`](https://pkg.go.dev/errors#AsType)):

```go
if apiErr, ok := errors.AsType[*api.Error](err); ok && apiErr.Code == "job_moved" {
	// The text was written for an Attempt that has ended; keep it for a retry.
}
```

- **Export one only when callers must match it** ([Uber: Error Types](https://github.com/uber-go/guide/blob/master/style.md#error-types)). Start with a sentinel, and move to a type when a caller needs fields.
- **Never compare error strings.** `err.Error() == "..."` breaks the moment someone rewords a message ([Google best practices](https://google.github.io/styleguide/go/best-practices#error-structure)).

### 8.5 Handle an error once

Either return an error or deal with it (print it, retry, fall back), not both. Logging and returning the same error prints it twice ([Uber: Handle Errors Once](https://github.com/uber-go/guide/blob/master/style.md#handle-errors-once), [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html)). In the CLI, the one place that prints errors is the error handler `main` gives Fang (section 13.4); everything else returns them.

### 8.6 Don't drop errors

`v, _ := f()` tells the compiler to stop checking and tells the reader nothing ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#handle-errors)). Where ignoring one is right, such as the error from closing a response body you've finished reading, say so with a short comment or by calling it in a `defer` whose error can't be acted on. `errcheck` will flag the rest.

### 8.7 `panic`

`panic` is for bugs, not for failures a user can cause. A missing Job, an unreachable server or a malformed flag is an error value ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#dont-panic)). A `Must` helper that panics is fine only on values fixed at compile time, such as a regular expression literal, never on input ([Google decisions](https://google.github.io/styleguide/go/decisions#must-functions)). Don't `recover` to hide a crash.

### 8.8 Exit from `main`, once

Only `main` calls `os.Exit`, and it calls it in one place. Everything else returns an error up to it ([Uber: Exit in Main](https://github.com/uber-go/guide/blob/master/style.md#exit-in-main), [Exit Once](https://github.com/uber-go/guide/blob/master/style.md#exit-once)). `os.Exit` skips deferred calls, so a function that exits can't clean up after itself.

## 9. Concurrency, context and signals

Most of the CLI runs one request at a time, so most code starts no goroutines. The HTTP client, Bubble Tea under the spinners, and Huh's prompts run their own, and leave you nothing to manage. When you do start one:

- **Know how it stops and who waits for it** before writing `go` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#goroutine-lifetimes), [Uber: Don't fire-and-forget goroutines](https://github.com/uber-go/guide/blob/master/style.md#dont-fire-and-forget-goroutines)). It stops on a cancelled context, a closed channel or the end of its work; something waits for it.
- **`sync.WaitGroup.Go`** (Go 1.25) starts a goroutine and does the counting, replacing the `Add(1)` / `defer Done()` pair ([Go 1.25](https://go.dev/doc/go1.25#syncpkgsync)):

  ```go
  var wg sync.WaitGroup
  for _, j := range jobs {
  	wg.Go(func() { refresh(ctx, j) })
  }
  wg.Wait()
  ```

  Since Go 1.22 each loop iteration has its own variable, so the old `j := j` copy is gone ([loop variables](https://go.dev/blog/loopvar-preview)).
- **`golang.org/x/sync/errgroup`** is the choice when each goroutine can fail and the first failure should cancel the rest.
- **Share memory by communicating** where a channel fits naturally, and use a `sync.Mutex` for shared state where it doesn't ([Effective Go](https://go.dev/doc/effective_go#sharing)). A channel is usually unbuffered or has a buffer of one ([Uber: Channel Size](https://github.com/uber-go/guide/blob/master/style.md#channel-size-is-one-or-none)). The sender closes it, never the receiver.

### Context

- **Pass a context as an argument; don't store it in a struct** ([Contexts and structs](https://go.dev/blog/context-and-structs)).
- **Inside a command, the context is `cmd.Context()`**, which comes from `fang.Execute` (Cobra's `ExecuteContext` underneath). Don't make a fresh `context.Background()` in a command ([Google best practices](https://google.github.io/styleguide/go/best-practices#complex-command-line-interfaces)).
- **Call the `cancel` a `WithTimeout` or `WithCancel` returns**, with `defer cancel()`, even if the deadline would end it anyway.
- **Never pass `nil` as a context.** Use `context.TODO()` while you work out where one should come from.
- **`context.Value` is for request-scoped data that crosses APIs, not for passing parameters.** The CLI has no use for it.

### Ctrl+C

Pressing Ctrl+C sends `SIGINT`, which by default ends a Go program at once ([`os/signal`](https://pkg.go.dev/os/signal)). `fang.WithNotifySignal(os.Interrupt)` instead cancels the context every command receives, so a request in flight stops cleanly and `main` can choose the exit code. Shells report a program ended by a signal as 128 plus the signal's number ([Bash manual](https://www.gnu.org/software/bash/manual/html_node/Exit-Status.html)), so an interrupted `taiichi` exits with **130**, as the prototype did. Huh reports a cancelled prompt as `huh.ErrUserAborted`; turn that into the same exit.

The one place this rule must not reach is the editor. Git ignores `SIGINT` while the editor is open, because Ctrl+C there belongs to the editor, and re-raises it only if the editor itself died of it ([`editor.c`](https://github.com/git/git/blob/master/editor.c)). If the editor ran under the signal-cancelled context, `exec.CommandContext` would kill it mid-edit and the text would be lost. Run the editor with `exec.Command`, which no context can cancel, and check the context once it returns (section 12.2).

## 10. Talking to the server

### 10.1 The shape of a call

Every API method is `POST /v1/<method>` with a JSON body, to the placeholder host `taiichi` over the socket. HTTP/1.1 needs a host name in every request, and the server ignores it. Every parameter is required, and parameters that can be empty are sent as `null`. A successful answer is a JSON object with the same keys every time; an error is `{"error": {"code", "message", "details"}}`.

The API has a small, fixed set of methods, so one `Call` function handles transport, encoding and errors, and a thin typed method per API method builds its parameters and result:

```go
// Package api is the CLI's client for the Taiichi server's HTTP API.
package api

// Client calls the server's API over its Unix domain socket. A Client is safe
// for concurrent use.
type Client struct {
	socket string
	http   *http.Client
}

// New returns a Client for the server listening on socket.
func New(socket string) *Client {
	var d net.Dialer
	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return d.DialContext(ctx, "unix", socket)
		},
	}
	return &Client{socket: socket, http: &http.Client{Transport: tr}}
}

// Call posts params to method and decodes a successful result into result.
// A refusal from the server comes back as an [*Error].
func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	body, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encode %s parameters: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://taiichi/v1/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
			return &UnreachableError{Socket: c.socket, Err: err}
		}
		return fmt.Errorf("call %s: %w", method, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read %s response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return decodeError(method, resp, data)
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("decode %s result: %w", method, err)
	}
	return nil
}

// Submit sends a Stage Output for the Attempt it was written for.
func (c *Client) Submit(ctx context.Context, p SubmitParams) (*SubmitResult, error) {
	var out SubmitResult
	if err := c.Call(ctx, "submit", p, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
```

The rules this follows:

- **Build one `http.Client` and reuse it.** Clients and transports are safe for concurrent use and "should only be created once and re-used" ([`net/http`](https://pkg.go.dev/net/http#pkg-overview)).
- **Dial the socket in `Transport.DialContext`**, ignoring the network and address HTTP asks for. Everything else about HTTP, such as keep-alive and cancellation, then works unchanged.
- **Every request carries the command's context** through `NewRequestWithContext`, so Ctrl+C cancels it.
- **Don't set `http.Client.Timeout`.** Every API call is synchronous and waits for Git and Forgejo work on the server, including a push, so no fixed limit is right for all of them. The context is how a call stops.
- **Always close the response body** ([`net/http`](https://pkg.go.dev/net/http#pkg-overview)). The `bodyclose` linter checks it.
- **Use `http.MethodPost` and `http.StatusOK`**, not `"POST"` and `200`.

### 10.2 When the server isn't there

A missing socket file fails to dial with `ENOENT`, and a socket file left behind by a crashed server fails with `ECONNREFUSED`. Both mean the same to the Operator, so both become one error type whose message says what to do, naming the path:

```go
// UnreachableError reports that no server answered on the socket.
type UnreachableError struct {
	Socket string
	Err    error
}

func (e *UnreachableError) Error() string {
	return fmt.Sprintf("can't reach the Taiichi server at %s; is it running?", e.Socket)
}

func (e *UnreachableError) Unwrap() error { return e.Err }
```

### 10.3 Server errors

Decode the error body into a type, and keep the HTTP status beside it. The CLI branches only on `Code`, and prints `Message` exactly as the server wrote it, because the server writes messages for people and is the only place that knows the rules behind them.

```go
// Error is a refusal from the server. Programs branch on Code; Message is
// written for people and is printed as it is.
type Error struct {
	Status  int            `json:"-"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func (e *Error) Error() string { return e.Message }
```

A response that isn't an error body (a proxy page, a crash) becomes an ordinary error naming the method and the HTTP status. `404 unsupported_version` means the server is older or newer than this CLI; its message already tells the Operator to restart it.

### 10.4 Finding the socket

The CLI and the server each work out the socket path the same way, so the CLI never reads the server's config: `TAIICHI_SOCKET` if set, else `$XDG_RUNTIME_DIR/taiichi/taiichi.sock`, else `~/.local/state/taiichi/taiichi.sock`. Take the environment as a function rather than calling `os.Getenv` inside, so tests can pass a map and still run in parallel, which `t.Setenv` forbids ([Mat Ryer](https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/), [`testing.T.Setenv`](https://pkg.go.dev/testing#T.Setenv)):

```go
// SocketPath returns where the server listens, from the environment getenv
// reads.
func SocketPath(getenv func(string) string) (string, error) {
	if p := getenv("TAIICHI_SOCKET"); p != "" {
		return p, nil
	}
	if dir := getenv("XDG_RUNTIME_DIR"); dir != "" {
		return filepath.Join(dir, "taiichi", "taiichi.sock"), nil
	}
	home := getenv("HOME")
	if home == "" {
		return "", errors.New("can't find the socket: HOME is not set")
	}
	return filepath.Join(home, ".local", "state", "taiichi", "taiichi.sock"), nil
}
```

`main` passes `os.Getenv`.

## 11. JSON

### 11.1 Which package

Go 1.27 added `encoding/json/v2` and `encoding/json/jsontext` to the standard library, and reimplemented the original `encoding/json` on top of them, keeping its behaviour ([Go 1.27](https://go.dev/doc/go1.27#encoding_json_v2)). New code can choose either. The two differ in ways that matter to this API ([`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2)):

| | `encoding/json` (v1) | `encoding/json/v2` |
|---|---|---|
| Nil slice or map | encodes as `null` | encodes as `[]` or `{}` |
| Field name matching on decode | case-insensitive | exact (snake_case keys stay exact) |
| Duplicate keys, invalid UTF-8 | accepted | rejected |
| Unknown keys | ignored unless `DisallowUnknownFields` | ignored unless `RejectUnknownMembers` |

**Provisionally**, use `encoding/json/v2` (`import "encoding/json/v2"`, still package `json`). Its stricter, exact defaults match an API that is strict about its own shapes, and a nil list becoming `[]` means a list parameter can never turn into `null` by accident.

### 11.2 Requests: every key, every time

The server rejects a missing parameter as firmly as an unknown one, so a request struct encodes every field on every call:

- **Tag every field with its exact snake_case key** (`json:"git_dir"`). Tags keep a Go rename from changing the wire format, and the API's keys aren't Go's MixedCaps ([Uber: Use field tags](https://github.com/uber-go/guide/blob/master/style.md#use-field-tags-in-marshaled-structs)).
- **No `omitempty` or `omitzero` on request fields.** Either one drops the key, and a dropped key is a `400 bad_request`.
- **A parameter that can be `null` is a pointer**, and `nil` encodes as `null`. Since Go 1.26, `new` takes an expression, so a pointer to a value needs no helper ([Go 1.26](https://go.dev/doc/go1.26#language)):

```go
// SubmitParams are the parameters of the submit method.
type SubmitParams struct {
	Job      int      `json:"job"`
	Stage    Stage    `json:"stage"`
	Attempt  int      `json:"attempt"`
	Body     string   `json:"body"`
	Verdict  *Verdict `json:"verdict"`
	Title    *string  `json:"title"`
	Unjudged bool     `json:"unjudged"`
}

p := api.SubmitParams{
	Job: 7, Stage: api.StageEvaluate, Attempt: 2, Body: body,
	Verdict: new(api.VerdictPass),
}
// {"job":7,"stage":"evaluate","attempt":2,"body":"…","verdict":"pass","title":null,"unjudged":false}
```

### 11.3 Results: tolerate new keys

The API promises stable shapes and wraps lists (`{"jobs": [...]}`) so that keys can be added later. A long-running server can also be newer than the CLI talking to it. So **don't reject unknown keys when decoding results**: a CLI that refused every new key would break on every server upgrade. Decode into a struct with the keys the CLI uses.

- **Use a pointer for a result field only when `null` and the zero value mean different things.** `worktree` is `null` once removed, and an empty path is never valid, so a `string` would do; `ended_at` is `null` while the Job is active, and a zero `time.Time` reads as "ended in year 1", so it's a `*time.Time`. Choose deliberately per field.
- **Ticket IDs are strings** (`"42"`), even though Forgejo's are numbers, so later Trackers with IDs like `"ENG-42"` fit.
- **`meta` and `details` vary by kind and code.** Decode them as `jsontext.Value` (raw JSON) or `map[string]any`, and turn them into typed values in the one function that needs them.

### 11.4 Custom encoding

Implement `MarshalJSON` and `UnmarshalJSON` (or v2's `MarshalJSONTo` and `UnmarshalJSONFrom`) only for a type whose wire form differs from its Go form. Plain string types such as `Position` need neither.

## 12. Running Git, and the editor

### 12.1 Git

The CLI asks Git where it is: `git rev-parse --path-format=absolute --git-common-dir` for the Git common dir that identifies the Repo, and `git rev-parse --show-toplevel` for the worktree that identifies the Job.

```go
// CommonDir returns the absolute path of the Git common dir shared by every
// worktree of the clone that contains dir.
func CommonDir(ctx context.Context, dir string) (string, error) {
	return run(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

func run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			msg := strings.TrimSpace(stderr.String())
			if exitErr.ExitCode() == 128 && strings.Contains(msg, "not a git repository") {
				return "", ErrNotARepository
			}
			return "", fmt.Errorf("git %s: %s", args[0], msg)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}
```

- **No shell.** `os/exec` "intentionally does not invoke the system shell" ([`os/exec`](https://pkg.go.dev/os/exec)). Pass each argument separately, so a path with spaces or a branch name with odd characters can never be read as shell syntax.
- **`exec.CommandContext`**, so a cancelled command stops its child too.
- **Capture stderr** and put it in the error: Git's message is what tells the Operator what went wrong. `Output` keeps a trimmed copy in `ExitError.Stderr` only when you leave `Cmd.Stderr` unset.
- **Set `cmd.Dir`** instead of changing the process's working directory.
- **Outside a Git repository is not a failure for every command.** `status` sends `git_dir: null` there. Turn Git's "not a git repository" into a sentinel, and let each command decide.
- **`--path-format=absolute` needs Git 2.31 or later.** Say so in the error if an older Git rejects it.

### 12.2 The editor

`taiichi submit` opens the editor the way `git commit -v` does: an empty area to write in, a **scissors line**, and the Stage Input below it for reference. Everything from the scissors line down is cut before sending, and an empty text aborts.

- **Find the editor with `git var GIT_EDITOR`.** Git checks `$GIT_EDITOR`, then `core.editor`, then `$VISUAL`, then `$EDITOR`, then its built-in default ([`git var`](https://git-scm.com/docs/git-var)), so `taiichi` opens whatever `git commit` would.
- **Run it through `sh -c`, the one exception to "no shell".** Git defines the editor value as "meant to be interpreted by the shell" ([`git var`](https://git-scm.com/docs/git-var)), so `code --wait` or `"$HOME/bin/vi"` must work. Pass the file as a positional argument (`"$@"`), never by pasting it into the command string.
- **Connect the terminal**: the editor needs the process's stdin, stdout and stderr.
- **Don't tie it to the signal-cancelled context** (section 9).

```go
// Scissors marks where the editable text ends. Everything from this line down
// is cut before the text is sent.
const Scissors = "# ------------------------ >8 ------------------------"

// Run opens path in editor and waits for it to exit.
func Run(editor, path string) error {
	// Git treats the editor as a shell command, so run it through sh and pass
	// the path as an argument, never as part of the command string.
	cmd := exec.Command("sh", "-c", editor+` "$@"`, editor, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor %s: %w", editor, err)
	}
	return nil
}

// Cut returns text up to the scissors line, with surrounding space trimmed.
func Cut(text string) string {
	body, _, _ := strings.Cut(text, Scissors)
	return strings.TrimSpace(body)
}
```

Check what you can before the editor opens, so no writing is lost to a mistake the CLI could have caught: `--pass` or `--fail` against the Job's Stage, and whether there's a terminal at all. If a submission fails after the editor closes, save the text under `$XDG_STATE_HOME/taiichi/submit/` and print the `taiichi submit <job> -f <path>` command that retries it.

## 13. Commands with Cobra and Fang

[Cobra](https://github.com/spf13/cobra) parses commands, flags and arguments, generates help and shell completion, and underlies `kubectl`, `gh` and most other large Go CLIs. [Fang](https://github.com/charmbracelet/fang) wraps a Cobra root to style help and errors, add `--version` and a `completion` command, and keep usage text out of runtime errors.

### 13.1 Building the tree

Cobra's generator puts each command in a package-level variable and wires flags in `init` functions ([Cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md)). Don't follow it: package-level state and `init` make commands hard to test and to build twice ([Uber: Avoid Mutable Globals](https://github.com/uber-go/guide/blob/master/style.md#avoid-mutable-globals), [Avoid `init()`](https://github.com/uber-go/guide/blob/master/style.md#avoid-init)). Build each command in a function that returns it, and keep its flags in local variables the closure captures:

```go
// NewRoot returns the taiichi command with every subcommand attached.
func NewRoot(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "taiichi",
		Short: "Move Jobs through the Pipeline, from Pickup to Done",
		// Fang sets these too, but tests run the tree without Fang.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newStatusCmd(deps), newPickupCmd(deps), newSubmitCmd(deps) /* ... */)
	return root
}

func newAbandonCmd(deps Deps) *cobra.Command {
	var yes, discard bool
	cmd := &cobra.Command{
		Use:               "abandon [job]",
		Short:             "End the Job as Abandoned",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeJobs(deps),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			job, err := resolveJob(ctx, deps, args)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := render.Confirm(fmt.Sprintf("Abandon Job %d?", job), "abandon", "--yes")
				if err != nil || !ok {
					return err
				}
			}
			j, err := deps.Client.Abandon(ctx, api.AbandonParams{Job: job, Discard: discard})
			if err != nil {
				return err
			}
			return render.JobSummary(cmd.OutOrStdout(), j)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "abandon without asking")
	cmd.Flags().BoolVar(&discard, "discard", false, "remove the worktree even with uncommitted changes")
	return cmd
}
```

- **`RunE`, never `Run`**, so failures return as errors and reach the one error handler ([`cobra.Command`](https://pkg.go.dev/github.com/spf13/cobra#Command)).
- **Set `SilenceUsage` and `SilenceErrors` on the root yourself.** Fang sets both when it runs, but a test that executes the tree directly would otherwise print the whole usage text after every error.
- **Validate positional arguments with `Args`**: `cobra.NoArgs`, `cobra.ExactArgs(1)`, `cobra.RangeArgs(1, 2)`, `cobra.MatchAll(...)`. Cobra then reports a wrong count before `RunE` runs.
- **Pass dependencies in.** A small `Deps` struct (the API client, the environment, the streams) handed to every constructor replaces globals, and lets a test swap in a client pointed at a test server.
- **One file per command** in `internal/command/`.

### 13.2 The Job-last grammar

Each command is a single verb, and the Job number goes last and is optional inside a Job's worktree (`taiichi cat build 7 -n 2`). Cobra sees these as ordinary positional arguments. Resolve the Job in one shared function: an explicit number always wins; otherwise ask Git for the worktree's top level and ask the server's `locate` method which Job owns it. Never read the number out of the branch name, which would tie the CLI to one Tracker's naming rule.

Parse a Job number with `strconv.Atoi` and refuse anything else, including `#42`: the shell reads a word starting with `#` as a comment, so the CLI would never see it.

### 13.3 Flags

- **Flag names are lowercase words joined by hyphens** (`--ready-label`, `--forge-url`), the usual form for long options and the one the spec's command list uses. Google's guide asks for snake case, but for its own flag package, defined only in `main` ([Google decisions](https://google.github.io/styleguide/go/decisions#flags)).
- **Repeatable flags use `StringArrayVar`**, which keeps each value whole. `StringSliceVar` also splits on commas, and a label could contain one.
- **Check combinations in `RunE` when Cobra's message would confuse.** Cobra's `MarkFlagsMutuallyExclusive("pass", "fail")` produces "If any flags in the group [pass fail] are set none of the others can be", which the prototype replaced with its own "pass exactly one of --pass or --fail".
- **Every prompt has a flag that answers it** (`--yes`), so every command can run without a terminal ([CLI Guidelines](https://clig.dev/#interactivity)).

### 13.4 Running it: `main` and Fang

```go
// Taiichi is the command-line client for the Taiichi server.
package main

func main() {
	os.Exit(run())
}

func run() int {
	deps, err := command.NewDeps(os.Getenv, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = fang.Execute(context.Background(), command.NewRoot(deps),
		fang.WithNotifySignal(os.Interrupt),
		fang.WithErrorHandler(render.Error),
		fang.WithoutManpage(),
	)
	switch {
	case err == nil:
		return 0
	case errors.Is(err, render.ErrAborted), errors.Is(err, context.Canceled):
		return 130
	default:
		return 1
	}
}
```

- **Replace Fang's error handler.** The default one capitalises the first letter of the message and appends a full stop, so a server message comes out altered, and an error quoting user input changes that input (`"x" isn't a Job number` printed as `"X" isn't a Job number.`). Off a terminal it still prints its boxed layout, without the colour. Taiichi's handler prints the message as it is:

  ```go
  // Error prints err for the Operator. Fang calls it with stderr, wrapped in a
  // writer that strips styling the terminal can't show.
  func Error(w io.Writer, _ fang.Styles, err error) {
  	switch apiErr, ok := errors.AsType[*api.Error](err); {
  	case errors.Is(err, ErrAborted), errors.Is(err, context.Canceled):
  		fmt.Fprintln(w, "Aborted.")
  	case ok:
  		fmt.Fprintln(w, errorStyle.Render(apiErr.Message))
  		fmt.Fprintln(w, codeStyle.Render(apiErr.Code))
  	default:
  		fmt.Fprintln(w, errorStyle.Render(err.Error()))
  	}
  }
  ```

  It finds the server's error with `errors.AsType` and prints its `Message`, not `err.Error()`. That way a layer that wraps an `api.Error` with `%w` on its way up adds nothing to what the Operator reads. An interrupt prints "Aborted." rather than Go's "context canceled".
- **Exit codes**: 0 for success, 1 for any error, 130 for an interrupt or a cancelled prompt.
- **Fang asks the terminal for its background colour** each time it renders help or an error, and on a terminal that never answers it waits about four seconds. Real terminal emulators answer in milliseconds; `script`, some CI runners and odd PTYs don't. Set the colour scheme explicitly with `fang.WithColorSchemeFunc`, or bound the query with a short timeout.
- **`--version`**: pass the version to Fang explicitly with `fang.WithVersion` rather than relying on what Fang works out for itself. Since Go 1.24, `go build` records the version from Git tags and commits in the build information, which `runtime/debug.ReadBuildInfo` reads ([Go 1.24](https://go.dev/doc/go1.24#go-command)). Where the version comes from is open (section 19).

### 13.5 Completion

Fang's `completion` command emits scripts for bash, zsh, fish and PowerShell. Dynamic completions are `ValidArgsFunction`s of type `cobra.CompletionFunc`, returning `[]cobra.Completion`:

```go
func completeJobs(deps Deps) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		jobs, err := deps.Client.Jobs(cmd.Context(), api.JobsParams{GitDir: deps.GitDir})
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var out []cobra.Completion
		for _, j := range jobs.Jobs {
			out = append(out, cobra.CompletionWithDesc(strconv.Itoa(j.Number), j.Ticket.Title))
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
```

- **Return `ShellCompDirectiveNoFileComp`** wherever file names make no sense, or the shell falls back to completing files.
- **Never print to stdout in a completion function**: the shell reads stdout as the list of choices ([Cobra completions](https://github.com/spf13/cobra/blob/main/site/content/completions/_index.md)). Use `cobra.CompDebugln` for tracing.
- **Test with the hidden command**: `taiichi __complete cat build ''`.

## 14. Output for people

The CLI's output is for people today. A `--json` flag for programs is planned soon after v1, which is one more reason to keep fetching data and laying it out in separate functions: the `render` package takes API values and writes text, and the commands never format strings themselves.

### 14.1 Streams

- **Results go to stdout; everything else goes to stderr**: notes, warnings, spinners, prompts' explanations and errors ([CLI Guidelines](https://clig.dev/#the-basics)). `taiichi path 7 | xargs ls` must receive only the path.
- **Write through `cmd.OutOrStdout()` and `cmd.ErrOrStderr()`**, never `os.Stdout` or `fmt.Println`, so tests can capture both by calling `SetOut` and `SetErr`.
- **Rendering functions take an `io.Writer`**, so the same function writes to a terminal, a pipe or a test buffer.

### 14.2 Terminals and colour

What colour, glyphs and motion may carry, and what readers with low vision or a screen reader need, is in [`accessibility.md`](accessibility.md). This section covers the mechanics.

Whether stdout is a terminal decides how much decoration to print. Check it once, with `term.IsTerminal(f.Fd())` from `github.com/charmbracelet/x/term`, and pass the answer down rather than asking again in every function.

- **Print styled text through Lip Gloss's writers** (`lipgloss.Fprintln`, or a `colorprofile.NewWriter` around stderr). They detect what the output supports and reduce or strip the escape codes to match: all of them on a pipe or when `TERM=dumb`, colour only when `NO_COLOR` is set ([`colorprofile.Detect`](https://pkg.go.dev/github.com/charmbracelet/colorprofile#Detect)). That matches the `NO_COLOR` convention, which turns off colour but not bold or italic ([no-color.org](https://no-color.org/)), and nothing else in the CLI needs to look at those variables.
- **Off a terminal, print plain layouts, not stripped ones.** Stripping escapes leaves padding, borders and margins behind. Lip Gloss tables lose their borders and padding only when you remove them (six `Border…(false)` calls and a padding `StyleFunc` in the prototype), and must not be given a width on a pipe, or long cells wrap there too.
- **Background colour**: `lipgloss.HasDarkBackground(os.Stdin, os.Stdout)` asks the terminal, and can wait seconds for one that never answers. Ask at most once per invocation (wrap it in `sync.OnceValue`), and only when about to draw something that depends on it.

### 14.3 Markdown

Stage Outputs and Inputs are Markdown, rendered with Glamour on a terminal and printed raw anywhere else.

```go
// Markdown writes md to w: rendered for a terminal, unchanged for anything else.
func Markdown(w io.Writer, md string, tty bool, width int) error {
	if !tty {
		_, err := io.WriteString(w, md)
		return err
	}
	style := "light"
	if isDark() {
		style = "dark"
	}
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err != nil {
		return err
	}
	out, err := r.Render(md)
	if err != nil {
		return err
	}
	_, err = lipgloss.Fprint(w, out)
	return err
}

var isDark = sync.OnceValue(func() bool {
	return lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
})
```

Glamour v2 no longer detects the background itself, and always emits colour, so the caller picks the style and decides when not to call it at all ([Glamour](https://pkg.go.dev/charm.land/glamour/v2)).

### 14.4 Prompts and spinners

- **Check for a terminal before every prompt**, on stdin and stdout, and without one fail with a message naming the flag that skips the question: "abandon needs a terminal to ask on; pass --yes to skip the question". Huh can't be trusted to do it: with stdin redirected inside a terminal, it opens `/dev/tty` and prompts anyway.
- **Never require a prompt.** The flag that answers it exists first ([CLI Guidelines](https://clig.dev/#interactivity)).
- **A dangerous action defaults to "no"**: `abandon`'s confirmation starts with its value `false`, so pressing Enter keeps the Job.
- **Huh's standalone forms always draw the light theme**, because they never ask the terminal for its background ([huh#834](https://github.com/charmbracelet/huh/issues/834), open). Until it's fixed, give forms a theme function that uses `isDark` instead of its argument.
- **Spinners go to stderr, and only on a terminal.** `huh/spinner` starts one regardless, so check first. Ctrl+C makes a spinner's `Run` return `tea.ErrInterrupted`; turn it into `ErrAborted` rather than carrying on to print results.

```go
// ErrAborted reports that the Operator cancelled a prompt or a spinner.
var ErrAborted = errors.New("aborted")

// CanPrompt reports whether stdin and stdout are both terminals.
func CanPrompt() bool {
	return term.IsTerminal(os.Stdin.Fd()) && term.IsTerminal(os.Stdout.Fd())
}

// Confirm asks question for command and reports the answer. Without a
// terminal it fails with a message naming flag, which answers the question
// instead.
func Confirm(question, command, flag string) (bool, error) {
	if !CanPrompt() {
		return false, fmt.Errorf("%s needs a terminal to ask on; pass %s to skip the question", command, flag)
	}
	ok := false
	err := huh.NewConfirm().Title(question).Affirmative("Yes").Negative("No").Value(&ok).Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return false, ErrAborted
	}
	return ok, err
}
```

### 14.5 Messages

- **Write errors and notes for the Operator**: what happened, and what to do next ([CLI Guidelines](https://clig.dev/#errors)). "can't reach the Taiichi server at /run/user/1000/taiichi/taiichi.sock; is it running?" beats `dial unix ...: connect: no such file or directory`.
- **Quote user-supplied values with `%q`**, which makes empty strings and stray whitespace visible ([Google decisions](https://google.github.io/styleguide/go/decisions#use-q)).
- **No stack traces** for anything a user can cause.

### 14.6 The post-v1 TUI

A terminal UI built with Bubble Tea v2 is planned after v1. Nothing in v1 depends on it. When it comes, the rule that keeps it cheap is the one above: the API client and the rendering functions know nothing about the command line.

## 15. Testing

The standard `testing` package, plus `github.com/google/go-cmp/cmp` for comparing values, covers what the CLI needs.

### 15.1 Shape

- **Test files sit beside the code**, as `name_test.go`. Use package `api` to test internals, `api_test` to test only the exported API.
- **Table-driven tests with subtests** are the norm ([Google decisions](https://google.github.io/styleguide/go/decisions#table-driven-tests)). Each case gets a `name`, run with `t.Run`, so a failure names its case and `go test -run 'TestSubmit/job_moved'` reruns one.
- **Don't force a table** on cases that need different setup or conditional assertions; write separate tests ([Uber: Test Tables](https://github.com/uber-go/guide/blob/master/style.md#test-tables)).
- **`t.Parallel()`** in every test and subtest that shares no process state. Since Go 1.22, loop variables are per iteration, so subtests no longer need the `tt := tt` copy.
- **`t.Helper()`** first in every helper that can fail a test, so the failure points at the caller.
- **`t.TempDir()`** for scratch folders, **`t.Context()`** (Go 1.24) for a context that ends with the test, **`t.Cleanup`** for teardown ([Go 1.24](https://go.dev/doc/go1.24#testingpkgtesting)).
- **`t.Setenv` and `t.Chdir` forbid `t.Parallel`**, because they change the whole process ([`testing.T.Setenv`](https://pkg.go.dev/testing#T.Setenv)). Prefer passing a `getenv` function and a directory (sections 10.4, 12.1).

```go
func TestSocketPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "explicit", env: map[string]string{"TAIICHI_SOCKET": "/tmp/t.sock"}, want: "/tmp/t.sock"},
		{name: "runtime dir", env: map[string]string{"XDG_RUNTIME_DIR": "/run/user/1000"}, want: "/run/user/1000/taiichi/taiichi.sock"},
		{name: "home", env: map[string]string{"HOME": "/home/ngs"}, want: "/home/ngs/.local/state/taiichi/taiichi.sock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := api.SocketPath(func(k string) string { return tt.env[k] })
			if err != nil {
				t.Fatalf("SocketPath(%v) error: %v", tt.env, err)
			}
			if got != tt.want {
				t.Errorf("SocketPath(%v) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}
```

### 15.2 Failure messages

- **A failure should be diagnosable without reading the test**: name the function, its input, what it returned, and what was wanted, got before want: `SocketPath(%v) = %q, want %q` ([Google decisions](https://google.github.io/styleguide/go/decisions#useful-test-failures)).
- **`t.Error` to report and carry on; `t.Fatal` when nothing after it can mean anything**, such as a setup failure ([Google decisions](https://google.github.io/styleguide/go/decisions#t-error-vs-t-fatal)). Never call `t.Fatal` from a goroutine other than the test's own.
- **Compare whole values with `cmp.Diff`**, and print the diff with its direction:

  ```go
  if diff := cmp.Diff(want, got); diff != "" {
  	t.Errorf("Jobs() mismatch (-want +got):\n%s", diff)
  }
  ```

  Avoid `reflect.DeepEqual`, whose failures say only "not equal" ([Google decisions](https://google.github.io/styleguide/go/decisions#equality-comparison-and-diffs)).
- **No assertion libraries**, including homemade ones. Google's guide says not to create them ([Google decisions](https://google.github.io/styleguide/go/decisions#assertion-libraries)); `testify` is widely used elsewhere, and is an open question here (section 19).
- **Don't compare bytes you don't control**, such as exact JSON output; decode it and compare values ([Google decisions](https://google.github.io/styleguide/go/decisions#compare-stable-results)).

### 15.3 Testing against the server

Test the real `*api.Client` against a real HTTP server on a real Unix socket, rather than faking the client: Google recommends using the production client with a test server over hand-written mocks ([Google best practices](https://google.github.io/styleguide/go/best-practices#use-real-transports)). `httptest` serves on any listener:

```go
func newServer(t *testing.T, h http.Handler) (socket string) {
	t.Helper()
	socket = filepath.Join(t.TempDir(), "taiichi.sock")
	l, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(h)
	srv.Listener = l
	srv.Start()
	t.Cleanup(srv.Close)
	return socket
}
```

A handler that answers with canned JSON from `testdata/` exercises encoding, decoding and every error path, including a missing socket (`ENOENT`) and a leftover one whose listener closed (`ECONNREFUSED`).

Socket paths have a length limit of about a hundred bytes on Linux and macOS. `t.TempDir()` paths are usually well under it; if a test fails with "invalid argument" on `Listen`, a long `TMPDIR` is the cause.

### 15.4 Testing commands

Run the Cobra tree in-process: build it with `NewRoot`, point its dependencies at a test server, set its arguments and streams, and execute it.

```go
root := command.NewRoot(deps)
var stdout, stderr bytes.Buffer
root.SetOut(&stdout)
root.SetErr(&stderr)
root.SetArgs([]string{"path", "7"})
if err := root.ExecuteContext(t.Context()); err != nil {
	t.Fatalf("taiichi path 7: %v", err)
}
```

For a whole command's printed output, compare against a **golden file**, a `testdata/<name>.golden` holding the expected text, with an `-update` flag that rewrites the files when the change is intended:

```go
var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", name, diff)
	}
}
```

Golden output for a pipe is plain text and stable. Golden output with escape codes is brittle; test styled output by running it through a `colorprofile` writer set to strip, or don't golden it.

For end-to-end tests of the built binary, with real exit codes and real stdin, **`testscript`** from `github.com/rogpeppe/go-internal` runs small shell-like scripts in txtar files, each in a fresh work folder, with `exec`, `stdout`, `stderr`, `cmp` and `!` for "must fail" ([testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)). It's how the Go command tests itself. Whether Taiichi needs it beside in-process tests is open (section 19).

### 15.5 Examples

An `Example` function in a test file is compiled, run, checked against its `// Output:` comment, and shown in the package's documentation. Use one for any function whose use is easier to show than to describe, such as `editor.Cut`.

## 16. Logging

The CLI doesn't log. It prints results for people and errors through one handler, and the server keeps the history. Adding `log/slog` to a command-line tool with nothing to record would only add a second way to print to stderr.

If a debug trace becomes worth having (the requests sent and the time each took, say), use `log/slog` from the standard library with a `TextHandler` on stderr, switched on by a flag or environment variable, and `slog.DiscardHandler` (Go 1.24) otherwise. Pass the `*slog.Logger` in through the same `Deps` as everything else. That's an open question, not a plan (section 19).

## 17. Doc comments and code comments

### 17.1 Doc comments

Go turns the comment directly above a top-level declaration into its documentation, shown by `go doc`, editors and pkg.go.dev ([Go Doc Comments](https://go.dev/doc/comment)).

- **Every exported name gets a doc comment**, a full sentence that starts with the name: `// Cut returns text up to the scissors line, ...` ([Code Review Comments](https://go.dev/wiki/CodeReviewComments#doc-comments)). The first sentence is what tools show as the summary.
- **A package comment** starts with "Package name" and sits directly above the `package` line in one file. A command's starts with the program's name and describes what it does: `// Taiichi is the command-line client for the Taiichi server.` ([Go Doc Comments](https://go.dev/doc/comment#cmd)).
- **A type's comment says what one value represents**, and what its zero value means if that isn't obvious. Say if it's safe for concurrent use, since readers assume it isn't ([Go Doc Comments](https://go.dev/doc/comment#type)).
- **A function's comment says what it returns, or what it does** if it's called for its effect. One returning a `bool` "reports whether" ([Go Doc Comments](https://go.dev/doc/comment#func)).
- **Doc links** are bracketed names: `[Client.Call]`, `[*Error]`, `[api.Job]`. They render as links.
- **Don't describe the implementation** in a doc comment, and don't restate the name: `// Client is a client.` is worse than nothing.

### 17.2 Code comments

These are the house rules for every comment in the tree, not only Go's:

- **A comment says what the code does and, where the names and types don't show it, the compressed why.** The test for keeping one: would the next person to edit this break something they can't see from the code? If not, delete it.
- **A comment never references a document outside the code.** No `docs/adr/…`, no spec section, no ticket number, no bare "ADR 0001". Put the reason itself in the comment, in a line, and drop the pointer. The design record links to code; code never links back to it. A doc link to another Go symbol is fine.
- **Unexported functions almost never get a comment.** Name them well and let the name and types carry them. One line is justified only for a real trap the code can't show: an ordering dependency, a deliberate absence, a footgun.
- **If a comment explains a name, rename instead.** A comment translating a unit, a default or what a function returns means the name is wrong: `timeoutSeconds`, not `timeout // in seconds`.
- **An ordering-dependency comment states the dependency and its reason, and stops**: `// Check the Verdict before opening the editor, so no writing is lost.`
- **Wrap comment prose at 80 columns.** `gofmt` doesn't rewrap comments, and comments are read in side-by-side diffs.

## 18. Conventions to adopt for Taiichi

These are the local defaults. Follow them unless a comment at the point of divergence says why.

1. **One module in `cli/`**, binary built from `cmd/taiichi/`, everything else in `internal/`. No `pkg/`.
2. **`gofmt` and `goimports` on save; `go vet`, `golangci-lint run` and `go test -race ./...` before every commit.** `golangci-lint` pinned through `mise`; `govulncheck` as a module tool.
3. **Run `go fix ./...` after every toolchain upgrade**, and review the diff.
4. **Names follow Go and the glossary**: MixedCaps, initialisms in one case, no stutter, no `Get`, and domain nouns as `GLOSSARY.md` spells them, never a word from an _Avoid_ list.
5. **No `init`, no package-level mutable state.** Commands are built by functions; dependencies arrive in a `Deps` struct.
6. **Cobra for commands, with Fang for help, errors, `--version` and completion; Huh for prompts; Lip Gloss for tables and styling; Glamour for Markdown.**
7. **`RunE` only. `SilenceUsage` and `SilenceErrors` set on the root.** Output goes through `cmd.OutOrStdout()` and `cmd.ErrOrStderr()`.
8. **The Job number goes last** and is optional inside a worktree, resolved through `locate`, never from the branch name.
9. **The CLI holds no rules.** It never parses `taiichi.toml` or `server.toml`, and checks only what saves the Operator from losing work before a request: a missing Verdict before the editor opens, a missing terminal before a prompt.
10. **One `api.Client` per invocation**, a reused `http.Client` dialling the socket, `POST http://taiichi/v1/<method>`, every request under the command's context, no client timeout.
11. **Branch only on the server's error `code`; print its `message` as it is.** Our own Fang error handler prints it, taken out of the error chain with `errors.AsType`, so neither Fang nor a wrapping layer ever alters it.
12. **An unreachable server is one error type**, covering a missing socket and a refused connection, whose message names the socket path.
13. **Request structs encode every field**: exact snake_case tags, no `omitempty` or `omitzero`, pointers for parameters that can be `null`, `new(expr)` to fill them.
14. **Result decoding tolerates unknown keys.**
15. **Wire enums are string types with constants**; Ticket IDs are strings.
16. **`encoding/json/v2`** for all JSON (provisional).
17. **`git` runs without a shell**, through `exec.CommandContext`, with stderr captured into errors. **The editor is the one shell exception**: found with `git var GIT_EDITOR`, run as `sh -c '<editor> "$@"'`, outside the signal-cancelled context.
18. **Results to stdout, everything else to stderr.** Plain layouts and raw Markdown off a terminal; styled output only through Lip Gloss's writers.
19. **A terminal check before every prompt**, failing with a message that names `--yes` (or the flag that answers it). Spinners only on a terminal, on stderr.
20. **Exit 0 on success, 1 on error, 130 on interrupt or a cancelled prompt.** `os.Exit` in `main` only, once.
21. **Errors wrapped with `%w` where callers should inspect them**, with a short lowercase prefix naming what this layer was doing. `errors.Is` and `errors.AsType` to inspect them; never string comparison.
22. **Standard `testing` and `cmp.Diff`**: table-driven, parallel, got before want, a real client against an `httptest` server on a Unix socket, golden files for printed output.
23. **No logging in v1.**
24. **Doc comments on every exported name; code comments only where the next editor would otherwise break something; never a reference to `docs/`, a spec or a ticket.**

## 19. Open questions

Points where the community is split or the project hasn't decided. Decide each, fold the answer into section 18, and delete it here.

1. **Package layout inside `cli/`.** Section 1's split (`command`, `api`, `git`, `editor`, `render`) is a proposal. The main alternative is fewer packages: everything but `api` in one `internal/cli`. Fewer packages share unexported helpers freely; more make each dependency visible in imports.
2. **Module path.** `github.com/ngscheurich/taiichi/cli` follows the repository's `origin`. If the repository moves, every import path changes with it, so settle it before the code grows.
3. **`encoding/json/v2` or v1.** v2 is in the standard library from Go 1.27 and its defaults suit a strict API, but it's new, and v1 is what most examples and libraries still show. Choose one for the whole module.
4. **Hand-written or generated API types.** The API has no machine-readable schema, so types are written by hand from the spec today. If the server gains one (OpenAPI, or JSON Schema exported from its parameter declarations), generating Go types would keep both sides in step.
5. **`testify` or the standard library.** `testify` (`assert.Equal(t, want, got)`) is the most used third-party test library and reads shorter. Google's guide rejects assertion libraries, and `cmp.Diff` gives better failure output. This guide follows Google for now.
6. **In-process command tests, `testscript`, or both.** In-process Cobra tests are fast and need nothing extra; `testscript` tests the real binary, its exit codes and its streams, at the cost of a dependency and a second way to write tests.
7. **A `golangci-lint` profile.** Section 2's configuration is a starting point. Linters worth weighing: `gosec`, `gocritic`, `exhaustive` (every `switch` on a `Position` covers every Position), `wrapcheck`.
8. **Fang's background query.** The spec says to use a short timeout or set the colour scheme explicitly; nobody has yet confirmed which Fang option avoids the query without losing light and dark themes.
9. **Client-side error codes.** The prototype invented `no_terminal` and `editor_failed` for failures that never reach the server, and printed them like server codes. Decide whether the CLI has codes of its own, and if so where they're listed.
10. **The terminal check's strictness.** The check refuses when stdin is redirected even inside a terminal (`echo | taiichi abandon`), which Huh itself would allow by opening `/dev/tty`. That's what the spec asks for; confirm it's wanted.
11. **Logging.** Whether a `--debug` trace (requests, timings) is worth having, and if so whether it's a flag or an environment variable.
12. **Version stamping.** Whether `--version` reads Go's build information, or the build passes a version with `-ldflags "-X ..."`, and whether it also prints the server's version.

---

## Appendix A: Quick reference

```go
// Package api is the CLI's client for the Taiichi server's HTTP API.
package api

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Wire enums: a string type with constants.
type Stage string

const (
	StagePlan     Stage = "plan"
	StageBuild    Stage = "build"
	StageEvaluate Stage = "evaluate"
	StageShip     Stage = "ship"
)

type Verdict string

const (
	VerdictPass Verdict = "pass"
	VerdictFail Verdict = "fail"
)

// Request: every key, every time; nil pointers encode as null.
type SubmitParams struct {
	Job      int      `json:"job"`
	Stage    Stage    `json:"stage"`
	Attempt  int      `json:"attempt"`
	Body     string   `json:"body"`
	Verdict  *Verdict `json:"verdict"`
	Title    *string  `json:"title"`
	Unjudged bool     `json:"unjudged"`
}

// Result: decode the keys you use; unknown keys are ignored.
type Job struct {
	Number   int        `json:"number"`
	Position Position   `json:"position"`
	Branch   string     `json:"branch"`
	EndedAt  *time.Time `json:"ended_at"`
}

// Sentinel and typed errors.
var ErrNotARepository = errors.New("not inside a Git repository")

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Message }

type ShowParams struct {
	Job int `json:"job"`
}

// One thin method per API method; wrap with %w where callers should inspect.
func (c *Client) Show(ctx context.Context, job int) (*Job, error) {
	var out struct {
		Job Job `json:"job"`
	}
	if err := c.Call(ctx, "show", ShowParams{Job: job}, &out); err != nil {
		return nil, fmt.Errorf("show Job %d: %w", job, err)
	}
	return &out.Job, nil
}

// Inspect with errors.Is and errors.AsType (Go 1.26), never by string.
func isMissing(err error) bool {
	apiErr, ok := errors.AsType[*Error](err)
	return ok && apiErr.Code == "no_such_job"
}

// A pointer to a value with new(expr) (Go 1.26).
var p = SubmitParams{Job: 7, Stage: StageEvaluate, Attempt: 2, Verdict: new(VerdictPass)}
```

```go
// Goroutines with sync.WaitGroup.Go (Go 1.25).
var wg sync.WaitGroup
for _, j := range jobs {
	wg.Go(func() { refresh(ctx, j) })
}
wg.Wait()
```

| Situation | Reach for |
|---|---|
| Run `git` | `exec.CommandContext(ctx, "git", args...)`, `cmd.Dir`, stderr into a `bytes.Buffer` |
| Open the editor | `git var GIT_EDITOR`, then ``exec.Command("sh", "-c", editor+` "$@"`, editor, path)`` |
| Is this a terminal? | `term.IsTerminal(f.Fd())`, checked once |
| Print styled text | `lipgloss.Fprintln(w, ...)` |
| Print Markdown | Glamour on a terminal, `io.WriteString` elsewhere |
| Ask a question | terminal check, then `huh.NewConfirm()...Run()`; `huh.ErrUserAborted` → exit 130 |
| Compare in a test | `cmp.Diff(want, got)`, reported as `(-want +got)` |
| Scratch folder in a test | `t.TempDir()` |
| Context in a test | `t.Context()` |
| Environment in a test | a `getenv` function, not `t.Setenv` |

## Appendix B: Reading list

- [Effective Go](https://go.dev/doc/effective_go): the original account of idiomatic Go. Written in 2009 and not updated for generics or modules, but its core still holds.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): the Go team's short list of the mistakes reviewers flag most.
- [Go Doc Comments](https://go.dev/doc/comment): how to write documentation `go doc` and pkg.go.dev render well.
- [Google's Go Style Guide](https://google.github.io/styleguide/go/): the guide, the decisions and the best practices, the most thorough style reference published.
- [Uber's Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md): practical rules with good and bad examples side by side.
- [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html), Dave Cheney: naming, package design, API design and errors, argued from first principles.
- [Organizing a Go module](https://go.dev/doc/modules/layout): the Go team's word on directory layout.
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors): wrapping, `errors.Is`, and when not to wrap.
- [Go release notes](https://go.dev/doc/devel/release): read each one when the toolchain in `mise.toml` moves.
- [Command Line Interface Guidelines](https://clig.dev/): streams, exit codes, prompts, colour and errors for any CLI.
- [Cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) and [completions](https://github.com/spf13/cobra/blob/main/site/content/completions/_index.md).
- [Fang](https://pkg.go.dev/charm.land/fang/v2), [Huh](https://pkg.go.dev/charm.land/huh/v2), [Lip Gloss](https://pkg.go.dev/charm.land/lipgloss/v2) and [Glamour](https://pkg.go.dev/charm.land/glamour/v2) package docs. Read the source when the docs run out; the prototype's notes record where they did.
- [`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2): read the package doc's comparison with v1 before writing custom encoding.
- [testscript](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript): end-to-end tests for command-line programs.
