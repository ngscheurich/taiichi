# Writing Elixir in Taiichi

> A guide to writing idiomatic Elixir for the Taiichi server, distilled from the Elixir team's own guides ([naming conventions](https://hexdocs.pm/elixir/naming-conventions.html), the four anti-pattern pages for [code](https://hexdocs.pm/elixir/code-anti-patterns.html), [design](https://hexdocs.pm/elixir/design-anti-patterns.html), [processes](https://hexdocs.pm/elixir/process-anti-patterns.html) and [macros](https://hexdocs.pm/elixir/macro-anti-patterns.html), [writing documentation](https://hexdocs.pm/elixir/writing-documentation.html), and [gradual set-theoretic types](https://hexdocs.pm/elixir/gradual-set-theoretic-types.html)), the release notes for Elixir [1.18](https://elixir-lang.org/blog/2024/12/19/elixir-v1-18-0-released/), [1.19](https://elixir-lang.org/blog/2025/10/16/elixir-v1-19-0-released/) and [1.20](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released), the docs of the libraries the server is built on (Ecto, `ecto_sqlite3`, Plug, Bandit, Req, Mox), the [community style guide](https://github.com/christopheradams/elixir_style_guide), and long-form writing by [Saša Jurić](https://www.theerlangelist.com/article/spawn_or_not), [José Valim](https://dashbit.co/blog/mocks-and-explicit-contracts) and [Chris Keathley](https://keathley.io/blog/good-and-bad-elixir.html).

The Taiichi server is one long-running Elixir program that serves every Repo on the machine. It answers HTTP with JSON bodies over a Unix domain socket (Bandit and Plug), keeps all its data in one SQLite database through Ecto, runs one `GenServer` per active Job with a checker beside it, and finishes work in Git and on Forgejo through a hand-rolled outbox. The Tracker and the Forge are behaviours, so other kinds can be added later. The rules of the Pipeline live in `Taiichi.Jobs.Pipeline` as pure functions. Most of the Elixir-specific design pressure falls in three places: keeping the domain rules pure while processes own the writes, turning every failure into one of the API's error codes, and testing all of it against a database that allows one writer at a time.

The guide is written for whoever writes the server next, person or coding agent. It assumes you know Elixir, OTP and Ecto, so it spends its words on conventions and reasons, not on teaching the language. Read sections 1 to 5 before your first module. Then jump to whichever layer your task touches, and read section 18 for the conventions this project has adopted and section 19 for the questions it hasn't settled. Where this guide and the Elixir team's guides disagree, the Elixir team wins, and this guide should be fixed.

Versions this guide was checked against: Elixir 1.20.4 on Erlang/OTP 29, Ecto 3.14, `ecto_sqlite3` 0.26, Plug 1.20, Bandit 1.13, Req 0.7, Mox 1.3 and Credo 1.7.

---

## 1. Project layout

The server's Mix app lives in `server/`, beside the Go CLI in `cli/`. The root of the repository holds only what both share: `docs/`, `GLOSSARY.md` and `mise.toml`, which pins both toolchains (`mise install` matches them).

```
server/
  mix.exs
  mix.lock                      # committed: this is an application, so the lock decides the build
  .formatter.exs
  config/
    config.exs                  # compile-time wiring only (section 14)
    test.exs
  lib/
    taiichi/
      application.ex            # the supervision tree, in startup order
      repos.ex                  # Taiichi.Repos: the Repo aggregate's public API
      repos/                    # its schemas and store code
      jobs.ex                   # Taiichi.Jobs: the Job aggregate's public API
      jobs/
        job.ex  attempt.ex  document.ex  ...   # Ecto schemas
        pipeline.ex             # pure rules
        server.ex               # the process for one Job
        checker.ex              # the checker for one Job
        store.ex                # the only Jobs code that calls Repo
      outbox.ex                 # the outbox sender
      tracker.ex                # the Tracker behaviour
      tracker/forgejo.ex
      forge.ex                  # the Forge behaviour
      forge/forgejo.ex
      git.ex
      api/
        router.ex               # Taiichi.API.Router
      repo.ex                   # Taiichi.Repo, the Ecto repo
  priv/
    repo/migrations/
  test/
    test_helper.exs
    support/                    # test-only modules, compiled in :test
    taiichi/...                 # mirrors lib/
```

Two names collide here, so keep them apart in prose and code: a **Repo** (capitalised, the domain term) is a Git repository that ran `taiichi init`, and `Taiichi.Repo` is the Ecto repo, the database handle. The domain's modules are `Taiichi.Repos` and `Taiichi.Repos.Repo`.

One module per file, and the file path follows the module name in `snake_case`: `Taiichi.Jobs.Pipeline` lives in `lib/taiichi/jobs/pipeline.ex` ([community style guide](https://github.com/christopheradams/elixir_style_guide#modules)). Every module starts with `Taiichi.`, so nothing we define can clash with a dependency's module, since the VM loads only one module of a given name ([namespace trespassing](https://hexdocs.pm/elixir/code-anti-patterns.html#namespace-trespassing)).

Common commands, run from `server/`:

| What | Command |
|---|---|
| Fetch dependencies | `mix deps.get` |
| Compile, failing on any warning | `mix compile --warnings-as-errors` |
| Run the server | `mix run --no-halt` |
| Interactive shell with the server running | `iex -S mix` |
| Test | `mix test` |
| Test, failing on any warning | `mix test --warnings-as-errors` |
| Rerun only the failures | `mix test --failed` |
| Format | `mix format` |
| Check formatting (CI) | `mix format --check-formatted` |
| Rewrite deprecated constructs | `mix format --migrate` |
| Find compile-time dependencies | `mix xref graph --label compile-connected` |
| Open a module's source | `mix source Taiichi.Jobs.Pipeline` |
| Lint (provisional, section 19) | `mix credo --strict` |

## 2. Formatting and the compiler as gates

`mix format` is not negotiable. It has few options on purpose, so every Elixir project looks alike and nobody argues about layout. Run it on save. CI runs `mix format --check-formatted`.

`server/.formatter.exs`:

```elixir
[
  import_deps: [:ecto, :ecto_sql, :plug],
  subdirectories: ["priv/*/migrations"],
  inputs: ["{mix,.formatter}.exs", "{config,lib,test}/**/*.{ex,exs}"]
]
```

- **`import_deps`** pulls in each library's `locals_without_parens`, so `field :stage, Ecto.Enum` and `plug :match` stay without parentheses ([`mix format`](https://hexdocs.pm/mix/Mix.Tasks.Format.html)).
- **Leave `line_length` at its default of 98.** The formatter wraps code; it doesn't wrap comment prose, which has its own 80-column rule (section 17).
- **`mix format --migrate`** rewrites constructs the language is retiring: `unless` becomes `if not`, charlists become `~c"..."`, and bitstring modifiers lose their parentheses ([1.18 release notes](https://elixir-lang.org/blog/2024/12/19/elixir-v1-18-0-released/)). It changes the code, not just its layout, so review the diff.

**Warnings are errors.** CI runs `mix compile --warnings-as-errors` and `mix test --warnings-as-errors`. This matters more than it used to. Since 1.20 the compiler infers types across every construct and warns about code that is certain to fail and about dead code ([1.20 release notes](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released)). Those findings arrive as warnings, so treating warnings as errors is what turns the type checker into a gate. Section 8 covers what it catches.

## 3. Naming

From the Elixir team's [naming conventions](https://hexdocs.pm/elixir/naming-conventions.html), which the compiler and standard library follow:

- **`snake_case`** for variables, functions, module attributes and file names. **`CamelCase`** for modules, with initialisms kept capitalised: `Taiichi.API`, `Taiichi.Git`, not `Taiichi.Api`.
- **A trailing `!`** marks the variant that raises instead of returning `{:error, _}`: `fetch/1` and `fetch!/1`. Write the tuple version first and build the bang on top of it. Invalid arguments raise whether or not the name has a bang.
- **A trailing `?`** marks a function that returns a boolean: `ended?/1`, `dirty?/1`.
- **An `is_` prefix** is only for checks that work in guards (`defguard is_stage(s) when s in [:plan, :build, :evaluate, :ship]`). Never both `is_` and `?`.
- **`get`, `fetch`, `fetch!`** mean what they mean in `Map`: `get` returns `nil` or a default, `fetch` returns `{:ok, value}` or `:error` (in Taiichi, `{:error, %Taiichi.Error{}}`, section 7), and `fetch!` raises.
- **`size` is constant time and `length` is linear.** Name your own functions the same way.
- **A leading `_`** marks an unused variable: `_conn`. Don't start function names with `_`.

Taiichi's own naming rules:

- **Name modules and functions with the glossary's terms.** `Attempt`, `Stage`, `Shipment`, `Report`, `Ticket Snapshot`: if GLOSSARY.md lists a word under _Avoid_, it doesn't appear in a module, function, field or atom either. A function that returns a Job's Position is `position/1`, never `status/1` or `state/1`.
- **No stutter.** Inside `Taiichi.Jobs`, the function is `pickup/2`, not `pickup_job/2`; callers read `Jobs.pickup(...)`.
- **Exceptions end in `Error`** (`Taiichi.Error`), and messages raised for developers are lowercase with no closing period ([community style guide](https://github.com/christopheradams/elixir_style_guide#exceptions)). Messages the Operator reads through the API are sentences (section 7).
- **Atoms for closed sets, written out in code.** Stages, Reports, Verdicts and error codes are atoms (`:build`, `:refused`, `:pass`, `:job_moved`). See section 6 for why they must never be built from input.

## 4. Modules: what goes at the top, and what doesn't

Keep the top of a module in a fixed order, so a reader finds things where they expect them ([community style guide](https://github.com/christopheradams/elixir_style_guide#module-attribute-ordering)): `@moduledoc`, `@behaviour`, `use`, `import`, `require`, `alias`, module attributes, `defstruct` or `schema`, `@type`s, `@callback`s, then functions, with public functions before private ones.

```elixir
defmodule Taiichi.Tracker.Forgejo do
  @moduledoc """
  Forgejo issues as the Tracker.
  """
  @behaviour Taiichi.Tracker

  alias Taiichi.Error
  alias Taiichi.Jobs.Job

  @impl true
  def check(%Job{} = job) do
    # ...
  end
end
```

- **`alias` freely; `import` rarely.** An alias keeps the module name visible at every call site. Import only what a module uses throughout, such as `Ecto.Query` in store code or `Plug.Conn` in the router, and prefer `import Mod, only: [...]` elsewhere.
- **`use` only for the frameworks that require it**: `use GenServer`, `use Ecto.Schema`, `use Plug.Router`, `use ExUnit.Case`. `use` can inject any code into the caller, so prefer `import` or `alias` when they're enough, and never write a `__using__` of our own unless a framework calls for it ([macro anti-patterns](https://hexdocs.pm/elixir/macro-anti-patterns.html#use-instead-of-import)).
- **No macros of our own** where a function would do. A macro that could be a function is harder to read, needs a `require`, and adds compile-time dependencies ([unnecessary macros](https://hexdocs.pm/elixir/macro-anti-patterns.html#unnecessary-macros)). `defguard` is the usual exception.
- **Refer to the current module as `__MODULE__`**, so a rename doesn't leave stale references.
- **Write module names out in full.** Never build them with `Module.concat/2` or string interpolation: the compiler can't track a dependency it can't see, so files stop recompiling when they should ([untracked dependencies](https://hexdocs.pm/elixir/macro-anti-patterns.html#untracked-compile-time-dependencies)). The Tracker and Forge implementations are looked up in a map of literal module names (section 12) for this reason.

### Public API modules and boundaries

Each aggregate has one public module, `Taiichi.Repos` and `Taiichi.Jobs`. Other code calls those, never the schemas' store code or a Job's process directly. `Taiichi.API` calls only `Repos` and `Jobs`. This is the "boundary" in domain-driven design: the HTTP layer and storage stay on either side of the rules, and a change inside `Jobs` can't ripple out unnoticed.

A function in a public module with only one caller is fine. Within private code, Keathley's advice applies: inline a helper with a single call site unless its name says something the body doesn't ([Good and bad Elixir](https://keathley.io/blog/good-and-bad-elixir.html)).

## 5. Functions: patterns, pipes, `with`, `case`

### Assertive code

Elixir code should say what it expects and crash when it doesn't get it. The Elixir team lists the failure to do so as three separate anti-patterns ([code anti-patterns](https://hexdocs.pm/elixir/code-anti-patterns.html)):

- **Pattern-match on what you expect.** Match `{:ok, value}` and `{:error, reason}` explicitly rather than catching everything with `_`, so a new return shape fails loudly instead of being swallowed.
- **Use `map.key` for keys that must be there**, and `map[:key]` only for keys that may be missing. `map[:key]` returns `nil` for a missing required key, and the `nil` travels far from its cause.
- **Use `and`, `or` and `not` when the operands are booleans.** They raise on a non-boolean, where `&&`, `||` and `!` accept anything.
- **Match in the function head only what you dispatch on**; destructure the rest in the body. A head that pulls out ten fields hides which ones choose the clause ([complex extractions](https://hexdocs.pm/elixir/code-anti-patterns.html#complex-extractions-in-clauses)).
- **Guard for what you want, not against what you don't**: `when is_binary(body)`, not `when not is_nil(body)` ([Keathley](https://keathley.io/blog/good-and-bad-elixir.html)).

```elixir
# Dispatch on the Stage and the Verdict; nothing else in the head.
def next_stage(:plan, nil), do: :build
def next_stage(:build, nil), do: :evaluate
def next_stage(:evaluate, :pass), do: :ship
def next_stage(:evaluate, :fail), do: :build
```

There is no catch-all clause: `next_stage(:ship, nil)` has no meaning, so it raises `FunctionClauseError`, and that is the right outcome for a programmer error.

### Pipes

- **Start a pipe with a plain value** and use it for a chain of two or more transformations ([community style guide](https://github.com/christopheradams/elixir_style_guide#pipe-operator)). A single `|>` is just a function call written backwards.
- **Don't pipe into `case`, and don't pipe the result of a side effect onward.** Bind the result to a well-named variable and branch on it. Keathley's point is that the function that called the side effect is the only one that knows what to do when it fails ([Good and bad Elixir](https://keathley.io/blog/good-and-bad-elixir.html)).

### `with`

`with` chains steps that each return `{:ok, _}` and stops at the first one that doesn't match ([`with/1`](https://hexdocs.pm/elixir/Kernel.SpecialForms.html#with/1)). Use it when any failure can fall through to the caller unchanged.

```elixir
def pickup(%Repo{} = repo, ticket_id) do
  with {:ok, settings} <- Repos.read_settings(repo),
       {:ok, ticket} <- fetch_ticket(settings, ticket_id),
       :ok <- check_eligible(ticket) do
    Store.save_pickup(repo, settings, ticket)
  end
end
```

- **Make each step return the error the caller should see.** Then the `with` needs no `else`. A long `else` funnels every failure into one place where you can no longer tell which step produced it, and the Elixir team lists it as an anti-pattern ([complex `else` clauses](https://hexdocs.pm/elixir/code-anti-patterns.html#complex-else-clauses-in-with)). Normalise errors in small private functions next to where they happen.
- **If the caller has to treat each failure differently, write `case`**, not `with` plus an `else` that tells them apart.

### `if`, `cond`, `case`

- **No `unless`.** The language plans to deprecate it, and `mix format --migrate` rewrites it to `if not`.
- **`cond` ends with `true ->`** when it needs a clause that always matches ([community style guide](https://github.com/christopheradams/elixir_style_guide#true-as-last-condition)).
- **Prefer function clauses over a `case` on an argument** when the function does nothing else. Prefer `case` over several functions when the branches share setup.
- **One function name, one purpose.** Multi-clause functions are for variations of one operation. `update/1` that updates a Job in one clause and a Repo in another is two functions sharing a name ([unrelated multi-clause function](https://hexdocs.pm/elixir/design-anti-patterns.html#unrelated-multi-clause-function)).

### Options and arguments

- **More than three or four parameters is a smell.** Group the ones that travel together into a struct or map ([long parameter list](https://hexdocs.pm/elixir/code-anti-patterns.html#long-parameter-list)).
- **No boolean flags that change what a function does.** Two flags that interact (`discard: true, force: true`) should be one atom option, or two functions ([boolean obsession](https://hexdocs.pm/elixir/design-anti-patterns.html#boolean-obsession)). The API keeps its boolean parameters, such as `abandon`'s `discard`, because they are wire format; turn them into atoms or separate functions at the API boundary.
- **An option never changes the return type.** If one mode returns a list and another a map, they are two functions ([alternative return types](https://hexdocs.pm/elixir/design-anti-patterns.html#alternative-return-types)).

## 6. Data: structs, atoms, maps and time

### Ecto schemas are the domain model

Taiichi doesn't keep a second set of domain structs beside its Ecto schemas: `%Taiichi.Jobs.Job{}` is both the row and the domain value, because two structs per concept would cost a translation layer for little gain. What keeps this honest is one rule: **only each aggregate's store code calls `Taiichi.Repo`**. Everything else, `Jobs.Pipeline` above all, takes schema structs and returns them unsaved, so its tests build structs by hand and never touch the database.

```elixir
defmodule Taiichi.Jobs.Attempt do
  @moduledoc """
  One try of a Job at a Stage.
  """
  use Ecto.Schema

  @type stage :: :plan | :build | :evaluate | :ship
  @type t :: %__MODULE__{}

  schema "attempts" do
    belongs_to :job, Taiichi.Jobs.Job
    field :seq, :integer
    field :stage, Ecto.Enum, values: [:plan, :build, :evaluate, :ship]
    field :number, :integer
    field :started_at, :utc_datetime_usec
  end
end
```

- **`Ecto.Enum` for closed sets.** The column stores the string (`"build"`), the struct holds the atom (`:build`), and Ecto refuses any other value.
- **Read `meta` through one small function per document kind.** A document's `meta` is a JSON column, so it comes back as a map with string keys. A function such as `Evaluation.verdict(document)` turns it into Elixir values once, at the edge, so no other code reads `document.meta["verdict"]`.
- **Keep structs under 32 fields.** Past 32, the VM stores a map differently and it uses more memory ([structs with 32 fields or more](https://hexdocs.pm/elixir/code-anti-patterns.html#structs-with-32-fields-or-more)). A Job is near no such limit today; nest rarely used fields if one grows.

### Atoms are never built from input

Atoms are never garbage-collected, so turning outside input into atoms can fill the VM's atom table and crash it ([dynamic atom creation](https://hexdocs.pm/elixir/code-anti-patterns.html#dynamic-atom-creation)). Never call `String.to_atom/1` on anything from the API, a TOML file, Forgejo or Git. Map known strings to atoms with function clauses, which also reject unknown values:

```elixir
defp parse_stage("plan"), do: {:ok, :plan}
defp parse_stage("build"), do: {:ok, :build}
defp parse_stage("evaluate"), do: {:ok, :evaluate}
defp parse_stage("ship"), do: {:ok, :ship}
defp parse_stage(other), do: {:error, Error.bad_request("stage", other)}
```

Decoded JSON keeps string keys for the same reason. Don't convert a decoded body's keys to atoms; read the keys you declared (section 11).

### Primitive obsession

When a string or integer carries structure, give it a struct and parse it once ([primitive obsession](https://hexdocs.pm/elixir/design-anti-patterns.html#primitive-obsession)). Candidates in Taiichi are the Forgejo location (instance URL plus `owner/repo`, compared after normalising) and the Ticket ID. A bare `"42"` passed through five functions invites someone to compare it with the integer `42`.

### Time

- **Store UTC with microseconds**: `:utc_datetime_usec` fields, filled from `DateTime.utc_now()`. The API sends RFC 3339 UTC, which `DateTime.to_iso8601/1` produces.
- **Pass "now" in as an argument** to pure functions (`Pipeline.next_attempt(current, attempts, verdict, now)`), so tests don't depend on the clock.

## 7. Errors

Elixir has two error channels, and the choice between them is the caller's need, not the severity ([design anti-patterns](https://hexdocs.pm/elixir/design-anti-patterns.html#exceptions-for-control-flow), [Keathley](https://keathley.io/blog/good-and-bad-elixir.html)):

- **Return `{:error, reason}` when the caller can do something about it.** Every failure the API answers with a code is in this group: `job_moved`, `worktree_dirty`, `forgejo_failed`.
- **Raise when nothing the caller does would help**: a programmer error, an invariant the code itself broke, data the server wrote that is now malformed. Let the process crash and its supervisor restart it.
- **Never use `try/rescue` for expected failures.** Call the non-raising function and `case` on its result. Bang functions are built on top of the tuple versions, never the other way round.

### One error type for the domain (provisional)

Every domain failure carries the same struct, so the API can render any of them without knowing where they came from:

```elixir
defmodule Taiichi.Error do
  @moduledoc """
  A domain error, carried in `{:error, error}` and rendered by the API.
  """

  @enforce_keys [:code, :message]
  defexception [:code, :message, details: nil]

  @type t :: %__MODULE__{code: atom(), message: String.t(), details: map() | nil}

  def new(code, message, details \\ nil) when is_atom(code) and is_binary(message) do
    %__MODULE__{code: code, message: message, details: details}
  end
end
```

- **`code`** is the atom the API sends as its `error.code` (`:job_moved`). Programs, including the CLI, branch only on it.
- **`message`** is a sentence for the Operator, written where the failure is understood, such as "No `forge.token` in taiichi.toml or taiichi.local.toml." The CLI prints it as it is.
- **`details`** carries data only for codes that need it, such as `job_moved`'s current Position.
- **It is an exception struct**, so the rare caller that wants to raise can `raise error`.

Keathley calls this "a common error type"; it is the most direct way to meet the API's rule that one error body shape covers every code. Its exact shape is still open (section 19).

### Constraint violations become domain errors

A broken unique index must never reach the API as a raw database error. Declare the constraint on the changeset, so Ecto returns `{:error, changeset}` instead of raising, then turn that changeset into a `Taiichi.Error` in store code:

```elixir
def pickup_changeset(job \\ %Job{}, attrs) do
  job
  |> cast(attrs, [:ticket_id, :tracker_url, :tracker_repo, :settings, :picked_up_at])
  |> validate_required([:ticket_id, :tracker_url, :tracker_repo, :settings, :picked_up_at])
  |> unique_constraint([:tracker_url, :tracker_repo, :ticket_id])
end
```

SQLite's error message names the columns, not the index, so `ecto_sqlite3` rebuilds the index name from the columns with Ecto's default rule (`jobs_tracker_url_tracker_repo_ticket_id_index`). **Never give a unique index a custom name**, or `unique_constraint/3` can't match it. The same adapter can't tell which foreign key failed, so `foreign_key_constraint/3` may not match either ([`ecto_sqlite3` limitations](https://hexdocs.pm/ecto_sqlite3/Ecto.Adapters.SQLite3.html)); check foreign keys yourself where the API owes a specific code.

## 8. Types

### What the compiler checks today

Elixir 1.20 infers types for every program without annotations and reports two things: code that is certain to fail at runtime, and dead code, such as a `case` clause that can never match ([gradual set-theoretic types](https://hexdocs.pm/elixir/gradual-set-theoretic-types.html)). It only reports what it can prove, so false positives are rare. It narrows types from patterns, guards and earlier clauses: after `is_map_key(job, :shipment)` it knows the key exists, and after a `case` clause matching `nil`, the next clause knows the value isn't `nil`.

A real example from writing this guide: a test helper `fn n -> send(pid, {:checked, n}) && %{state: :open} end` drew a warning that the conditional "will always evaluate to `{:checked, term()}`", because `send/2` returns its message, which is always truthy. That is the non-assertive truthiness of section 5, caught by the compiler.

You get the most from the checker by writing assertive code: match on structs (`%Job{} = job`) and on tagged tuples, use guards, and avoid catch-all clauses. Each pattern gives it a type to check against.

### Typespecs

The compiler doesn't check `@spec` today. Typespecs are Erlang's notation, which the Elixir team says isn't precise enough for the new system; typed structs and then type signatures are planned, after which typespecs are expected to move out of the language ([gradual set-theoretic types](https://hexdocs.pm/elixir/gradual-set-theoretic-types.html), [1.20 release notes](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released)). Until then, typespecs are documentation that Dialyzer can partly check.

Provisional policy, pending section 19:

- **`@callback` in every behaviour, with full types.** A behaviour is a contract, and the callback types are the contract's text.
- **`@type t` on every schema and struct**, so other specs can name it.
- **`@spec` on the public functions of `Taiichi.Repos`, `Taiichi.Jobs` and other public modules**, where a reader most needs to know what comes back. Not on private functions.
- **Spec the error shape precisely**: `{:ok, Job.t()} | {:error, Taiichi.Error.t()}`, never `{:error, term()}`.

## 9. Processes and OTP

### When a process, and when a module

A process is for a runtime concern: state that changes, work that runs concurrently, or failures that must be isolated. It is never for organising code ([GenServer docs](https://hexdocs.pm/elixir/GenServer.html), [code organisation by process](https://hexdocs.pm/elixir/process-anti-patterns.html#code-organization-by-process)). Saša Jurić's version: "Use functions and modules to separate thought concerns. Use processes to separate runtime concerns" ([To spawn, or not to spawn?](https://www.theerlangelist.com/article/spawn_or_not)).

Taiichi's processes each answer one runtime need:

| Process | How many | Why it's a process |
|---|---|---|
| `Jobs.Server` | one per active Job | Writes to one Job must happen one at a time; a process serialises them without locks, and one Job's crash can't touch another. |
| `Jobs.Checker` | one per active Job | It runs on a timer, independent of requests. |
| `Outbox` sender | one | Ended Jobs have no process, and Done still has work after its save. |
| Bandit's acceptors | Bandit's choice | One process per connection, so a slow request blocks no one else. |

Everything else is plain functions. `Jobs.Pipeline` is the clearest case: it decides moves and builds Stage Inputs, and it has no reason to run anywhere but in the caller. Saša's blackjack example has the same shape: a pure `Round` that returns instructions, driven by a `GenServer` that carries them out.

**Reads don't go through the Job's process.** `jobs`, `show`, `input`, `document` and `locate` read the database directly, so they never wait behind a slow push.

### One module owns each process's messages

All interaction with a process goes through public functions in the module that defines it. No other module calls `GenServer.call/3` on it or knows its message format ([scattered process interfaces](https://hexdocs.pm/elixir/process-anti-patterns.html#scattered-process-interfaces)). Keep the client functions and the callbacks in the same module unless one side grows large.

```elixir
defmodule Taiichi.Jobs.Server do
  @moduledoc """
  The process for one active Job, and the only writer of that Job.
  """
  use GenServer, restart: :transient

  require Logger

  @submit_timeout :timer.minutes(2)

  def start_link(number) do
    GenServer.start_link(__MODULE__, number, name: via(number))
  end

  @spec submit(pos_integer(), Submission.t()) :: {:ok, Job.t()} | {:error, Error.t()}
  def submit(number, %Submission{} = submission) do
    GenServer.call(via(number), {:submit, submission}, @submit_timeout)
  end

  defp via(number), do: {:via, Registry, {Taiichi.Jobs.Registry, number}}

  @impl true
  def init(number) do
    Logger.metadata(job: number)
    {:ok, %{number: number}}
  end

  @impl true
  def handle_call({:submit, submission}, _from, state) do
    {:reply, Store.submit(state.number, submission), state}
  end
end
```

- **`@impl true` on every callback**, so the compiler warns about a misspelt callback name.
- **Name processes through a `Registry`**, never with atoms built from the Job number. Names that come and go must not be atoms (section 6), and the `Registry` removes a key when its process dies ([`Registry`](https://hexdocs.pm/elixir/Registry.html)). The registry is local to the node, which is all Taiichi needs, since remote deployment is out of scope.
- **Set call timeouts on purpose.** `GenServer.call/3` gives up after 5 seconds by default, and the caller exits ([GenServer docs](https://hexdocs.pm/elixir/GenServer.html)). The caller here is the HTTP request's process, and `submit` at Ship pushes to Forgejo and waits for the outbox's first try, so the default is too short. Give each call that waits on Git or Forgejo an explicit, named timeout.
- **Send only what the process needs.** A message is copied in full into the receiving process, and so is everything a closure captures. Send the Job number and the Submission, not the `conn` ([sending unnecessary data](https://hexdocs.pm/elixir/process-anti-patterns.html#sending-unnecessary-data)).

### `init/1` stays fast

`start_link` blocks until `init/1` returns, and so does the supervisor starting the tree. Move slow or failure-prone setup into `handle_continue/2` by returning `{:ok, state, {:continue, term}}`; it runs before any other message ([GenServer docs](https://hexdocs.pm/elixir/GenServer.html)). The checker uses this to check at once on start without holding up its supervisor:

```elixir
@impl true
def init({number, opts}) do
  state = %{number: number, interval: Keyword.fetch!(opts, :interval)}
  {:ok, state, {:continue, :check}}
end

@impl true
def handle_continue(:check, state), do: {:noreply, check(state)}

@impl true
def handle_info(:check, state), do: {:noreply, check(state)}

defp check(state) do
  Server.report(state.number, check_job(state.number))
  Process.send_after(self(), :check, state.interval)
  state
end
```

The checker's timer is a plain `Process.send_after/3`, scheduled again after each check, so checks never overlap and a slow Forgejo only delays the next one. Oban's scheduled jobs were weighed for this and rejected.

### Supervision

Every process runs under a supervisor. A process spawned outside the tree can't be observed or restarted, and nothing guarantees its startup order or shutdown ([unsupervised processes](https://hexdocs.pm/elixir/process-anti-patterns.html#unsupervised-processes)). No bare `spawn/1` and no `Task.start/1` outside tests; use `Task.Supervisor` for one-off concurrent work.

Restart values ([`Supervisor`](https://hexdocs.pm/elixir/Supervisor.html)):

- **`:permanent`** (the default) for processes that should always run: the outbox sender, the `Registry`, Bandit.
- **`:transient`** for a Job's processes. They should come back after a crash, but stop for good when the Job ends: exiting with `:normal` or `{:shutdown, reason}` counts as success, so the supervisor leaves them stopped.
- **`:temporary`** for work that must not be retried by the supervisor, such as a one-off `Task`.

A Job's process and checker start together under the `DynamicSupervisor`. One way to give them a shared lifetime is a small per-Job supervisor, so ending the Job stops both and a crash of the Job's process restarts its checker (provisional, section 19):

```elixir
defmodule Taiichi.Jobs.JobSupervisor do
  @moduledoc false
  use Supervisor, restart: :transient

  def start_link({number, opts}), do: Supervisor.start_link(__MODULE__, {number, opts})

  @impl true
  def init({number, opts}) do
    children = [
      {Taiichi.Jobs.Server, number},
      {Taiichi.Jobs.Checker, {number, opts}}
    ]

    # The Checker reports to the Server, so the Server starts first.
    Supervisor.init(children, strategy: :rest_for_one)
  end
end
```

### Startup order is the order of the children

A supervisor starts its children in the order listed and stops them in reverse ([`Supervisor`](https://hexdocs.pm/elixir/Supervisor.html)). The server's startup sequence is therefore the order of `Taiichi.Application`'s children, and the socket opens last so no request sees a half-started server:

```elixir
children = [
  Taiichi.Repo,
  {Ecto.Migrator, repos: [Taiichi.Repo]},
  {Registry, keys: :unique, name: Taiichi.Jobs.Registry},
  {DynamicSupervisor, name: Taiichi.Jobs.Supervisor},
  Taiichi.Jobs.Restore,
  Taiichi.Outbox,
  {Bandit, plug: Taiichi.API.Router, ip: {:local, socket_path}, port: 0}
]

Supervisor.start_link(children, strategy: :one_for_one, name: Taiichi.Supervisor)
```

- **`{Ecto.Migrator, repos: [...]}`** runs pending migrations as a child, before anything that uses the tables. A failed migration crashes the child, which stops the server from starting.
- **`Taiichi.Jobs.Restore`** (an illustrative name) stands for a step that starts a process and checker for every Job that hasn't ended. Startup work that must finish before the next child starts belongs in a child whose `init/1` does the work and returns `:ignore`. A `Task` child would let the next child start before it finished.
- **Bandit on a Unix domain socket needs `port: 0`**; it fails without it ([Bandit](https://hexdocs.pm/bandit/Bandit.html)).
- **Configuration and socket checks happen before `Supervisor.start_link/3`**, in `start/2` itself: read `server.toml`, refuse unknown keys, and deal with a leftover socket file. Return `{:error, reason}` with a clear message rather than raising.

## 10. Ecto and SQLite

### Settings

ADR 0001 put the server's data in SQLite, which allows one write transaction at a time. Set every pragma explicitly, even where the adapter's default already matches, so the configuration says what the server relies on:

```elixir
config :taiichi, Taiichi.Repo,
  journal_mode: :wal,
  synchronous: :normal,
  foreign_keys: :on,
  busy_timeout: 5_000,
  default_transaction_mode: :immediate
```

- **`default_transaction_mode: :immediate`** makes every transaction take the write lock when it begins. With SQLite's default `DEFERRED` mode, a transaction that reads first and writes later can fail with `SQLITE_BUSY` when it tries to upgrade, instead of waiting out `busy_timeout` ([SQLite transactions](https://www.sqlite.org/lang_transaction.html)). Every transaction in Taiichi exists to write, so take the lock up front ([`ecto_sqlite3`](https://hexdocs.pm/ecto_sqlite3/Ecto.Adapters.SQLite3.html#module-transaction-mode)). The spec asks for `BEGIN IMMEDIATE` at Pickup; this setting gives it everywhere.
- **The database path is worked out at runtime**, from `$XDG_DATA_HOME`, because tests move the data folder by setting it. Compute it in `Taiichi.Repo`'s `init/2` callback, not in `config/config.exs`, which is evaluated at compile time.

### Transactions: one save per action

Each action that moves a Job has exactly one save: one transaction, with its outbox rows written inside it, and no transaction stays open while Taiichi waits on Git or Forgejo.

- **Use `Repo.transact/2`.** Ecto 3.13 added it and soft-deprecated `Repo.transaction/2` ([Ecto changelog](https://hexdocs.pm/ecto/changelog.html)). The function you pass returns `{:ok, result}` or `{:error, reason}`, and an `{:error, _}` rolls the transaction back and comes out unchanged, which composes with `with`:

  ```elixir
  def save_pickup(repo, settings, ticket) do
    Repo.transact(fn ->
      with {:ok, job} <- insert_job(repo, settings, ticket),
           {:ok, _snapshot} <- insert_snapshot(job, ticket),
           {:ok, _plan} <- insert_first_attempt(job),
           {:ok, _rows} <- Tracker.WriteBack.moved(nil, job) do
        {:ok, job}
      end
    end)
  end
  ```

- **Reach for `Ecto.Multi` only when the set of steps is built at runtime.** Ecto's docs call plain control flow inside `transact` "more straightforward" for everything else ([`Ecto.Multi`](https://hexdocs.pm/ecto/Ecto.Multi.html)).
- **Never call Git, Forgejo or anything slow inside the function.** SQLite has one writer, so a slow call there stalls every other Job's save.

### Migrations

- **Let `create table/2` make the primary key.** Ecto's default key, with `ecto_sqlite3`, is `INTEGER PRIMARY KEY AUTOINCREMENT`, so IDs are never reused, which the Job number relies on.
- **Use Ecto's default index names** (section 7).
- **Partial unique indexes** carry the `where:` option: `create unique_index(:jobs, [:tracker_url, :tracker_repo, :ticket_id], where: "ended_at IS NULL")` allows one active Job per Ticket.
- **Migrations are never edited once they've run anywhere**; add a new one. Each runs in its own transaction by default; leave that on.
- **SQLite can't alter much in place.** Changing a column's type or most constraints means creating a new table and copying the rows. Plan the schema change before writing the migration.

### Queries and loading

- **Queries live in store code**, next to the schemas they read, with `import Ecto.Query`.
- **Don't load what you don't need.** A document's `body` isn't loaded for listings: give the field `load_in_query: false` and select it only where asked for, or `select` the columns a listing uses.
- **`Repo.get/2` returns `nil`; turn that into `{:error, %Taiichi.Error{code: :no_such_job}}` in the store's `fetch/1`** instead of letting `nil` travel.

## 11. The HTTP API: Plug and Bandit

The API is plain remote procedure calls: every method is `POST /v1/<method>` with a JSON body, and every answer is a JSON object. The router is a `Plug.Router`, served by Bandit on the socket.

```elixir
defmodule Taiichi.API.Router do
  @moduledoc false
  use Plug.Router
  use Plug.ErrorHandler

  plug :match
  plug Plug.Parsers, parsers: [:json], pass: [], json_decoder: JSON
  plug :dispatch

  post "/v1/status" do
    with {:ok, params} <- Params.take(conn.body_params, git_dir: {:nullable, :string}),
         {:ok, result} <- Repos.status(params.git_dir) do
      Render.ok(conn, result)
    else
      {:error, %Error{} = error} -> Render.error(conn, error)
    end
  end

  # ... one route per method

  post "/v1/:method" do
    Render.error(conn, Error.new(:unknown_method, "This server has no method #{method}."))
  end

  match _ do
    Render.error(conn, Error.new(:unsupported_version, "This server speaks v1; restart it."))
  end

  @impl Plug.ErrorHandler
  def handle_errors(conn, %{reason: %Plug.Parsers.UnsupportedMediaTypeError{}}) do
    Render.error(conn, Error.new(:unsupported_media_type, "Send a JSON body."))
  end

  def handle_errors(conn, _failure) do
    Render.error(conn, Error.new(:internal, "Something went wrong on the server."))
  end
end
```

- **`plug :match`, then `Plug.Parsers`, then `plug :dispatch`.** The parser runs only for a request that matched a route ([`Plug.Router`](https://hexdocs.pm/plug/Plug.Router.html)).
- **`pass: []`** makes any content type other than JSON raise `UnsupportedMediaTypeError`, which the error handler turns into `415`. The parser's default size limit is 8,000,000 bytes, past which it raises `RequestTooLargeError` (`413`) ([`Plug.Parsers.JSON`](https://hexdocs.pm/plug/Plug.Parsers.JSON.html)). A body that isn't a JSON object arrives under a `"_json"` key; reject it as `400 bad_request`.
- **The catch-all `match _`** is required: without it, an unmatched path raises a `FunctionClauseError`. Put `post "/v1/:method"` just above it, so an unknown method under the right version is `unknown_method`, not `unsupported_version`. The sketch leaves out answering other HTTP methods with `405`.
- **`Plug.ErrorHandler` turns anything unexpected into `500 internal`** with the standard error body, never a stack trace. It still logs the original error. Never `use Plug.Debugger` in the server.
- **Routes stay thin.** A route takes its parameters, calls one function in `Taiichi.Repos` or `Taiichi.Jobs`, and renders the result. No rules in the router.

### Parameters are declared once per method

The API's parameters are exact: every one required, `null` where empty, and an unknown, missing or wrongly typed parameter is `400 bad_request` naming it. Write that check once, as a function that takes the decoded body and a declaration of each parameter's type, and call it at the top of every route. A schemaless changeset (`Ecto.Changeset.cast/4` on `{%{}, types}`) can do the casting ([`Ecto.Changeset`](https://hexdocs.pm/ecto/Ecto.Changeset.html)), but it treats a missing key and an unknown key differently from what the API needs, so the exactness checks around it are ours.

### Responses are built by hand

Build each response map explicitly in an API module, such as `Taiichi.API.JSON.job(job)`, and don't `@derive JSON.Encoder` on schemas. The API promises stable shapes, every key present with `null` where a field doesn't apply, and a derived encoder would change the API whenever someone added a field to a schema. The schema serves the domain; the response serves the CLI.

On a Unix domain socket, `conn.remote_ip` is `{:local, ""}`, which Plug's types don't expect. Nothing reads it.

## 12. Behaviours: the Tracker and the Forge

The core talks to the Tracker and the Forge through behaviours, so the rules stay the same when GitHub or Linear is added later. A behaviour is a module of `@callback`s; an implementation declares `@behaviour` and marks each callback `@impl true`, and the compiler warns if one is missing or misspelt.

```elixir
defmodule Taiichi.Tracker do
  @moduledoc """
  The interface every Tracker implements.
  """

  alias Taiichi.Jobs.Job

  @type check :: %{state: :open | :closed | :deleted, changed: [:title | :body | :labels]}

  @callback check(Job.t()) :: {:ok, check()} | {:error, Taiichi.Error.t()}

  @spec impl(Job.t()) :: module()
  def impl(%Job{settings: %{"tracker" => %{"type" => type}}}) do
    Map.fetch!(implementations(), type)
  end

  defp implementations do
    Application.get_env(:taiichi, :trackers, %{"forgejo" => Taiichi.Tracker.Forgejo})
  end
end
```

- **The implementation is chosen by the `type` in the Job's settings**, the copy taken at Pickup, never by a global. `Map.fetch!/2` raises on an unknown type, which `taiichi.toml` validation should already have refused.
- **The callback speaks in domain terms.** The Tracker interface takes Positions and returns Ticket states; labels, issue numbers and Forgejo's JSON stay inside `Tracker.Forgejo`.
- **The map of implementations can be replaced in tests** through the application environment (section 14), which is how a test puts a fake in front of the real Forgejo (section 16).
- **No behaviour without a reason.** The Tracker and Forge exist to admit other implementations; a module with one implementation and no plans for another stays a plain module.

## 13. The outside world: Forgejo and Git

### Forgejo over HTTP: Req (provisional)

Each Forgejo module builds its client from the Job's settings and the current token:

```elixir
defp client(settings, token) do
  Req.new(
    base_url: settings["url"],
    auth: {:bearer, token},
    retry: false,
    receive_timeout: :timer.seconds(15)
  )
  |> Req.merge(Application.get_env(:taiichi, :forgejo_req_options, []))
end
```

- **`retry: false`.** Taiichi already retries at the right level: a check runs again next interval, an outbox row is retried, and a command fails and the Operator runs it again. Req's own retries would multiply that, and its default retries only `GET` and `HEAD` anyway ([Req](https://hexdocs.pm/req/Req.html)).
- **Branch on the response yourself.** Req returns 4xx and 5xx as ordinary `{:ok, %Req.Response{}}`; a network failure is `{:error, %Req.TransportError{}}`. Turn each into a domain result or a `Taiichi.Error` in the Forgejo module, matching statuses explicitly (a 404 on an issue means deleted; a 404 on the repository means the token lost access).
- **Tokens are read fresh, never stored**, so they appear only in the request's `auth` and in no struct that might be logged.
- **No `Req.default_options/1`.** It is global; keep options in the module's own `Req.new/1` ([Req](https://hexdocs.pm/req/Req.html)).

### Git through `System.cmd/3`

```elixir
defp git(args, dir) do
  System.cmd("git", args,
    cd: dir,
    stderr_to_stdout: true,
    env: [{"GIT_TERMINAL_PROMPT", "0"}]
  )
end
```

- **Always `System.cmd/3` with a list of arguments, never `System.shell/2` or `:os.cmd/1`.** `cmd` passes each argument to Git unchanged, without a shell, so a branch name made from a Ticket's title can't inject a command ([`System.cmd/3`](https://hexdocs.pm/elixir/System.html#cmd/3)).
- **`GIT_TERMINAL_PROMPT=0`** makes Git fail instead of waiting forever for a password nobody can type.
- **Check the exit status every time.** `System.cmd/3` returns `{output, status}` and doesn't raise on a non-zero status.
- **`System.cmd/3` has no timeout.** A push that hangs holds the Job's process. How to bound it is open (section 19).
- **A retry checks the world, not a progress record.** Every Git and Forgejo step is written so that running it twice is safe, by asking whether it already happened: does the branch exist, is there already a pull request from this branch, did this account already post this comment.

## 14. Configuration

Taiichi reads its own configuration: `server.toml` for the server, and each Repo's `taiichi.toml` and `taiichi.local.toml`. Those are read by our code, with strict keys, at the moments the spec names. The Mix `config/` files are not the place for any of it.

- **`config/config.exs` and `config/test.exs` hold only wiring**: the Ecto repo's static settings, and the modules tests swap in (a fake Tracker, `Req.Test` options). Nothing a user sets goes there.
- **Read the application environment at runtime** with `Application.get_env/3`, not `Application.compile_env/3`, for anything tests change.
- **Pass values as arguments rather than reading globals deep in the code.** The application environment is global state, one value per key ([design anti-patterns](https://hexdocs.pm/elixir/design-anti-patterns.html#using-application-configuration-for-libraries)). That page is written for libraries, but the reason, that a global can't vary per caller, applies to Taiichi, where every Job runs under its own copy of its settings.
- **Unknown keys are errors**, in every TOML file. A TOML parser gives back a map; checking it against the known keys is ours to write, once, in the module that reads that file.
- **Hide tokens from `inspect/2`.** Any struct that holds a token derives `@derive {Inspect, except: [:token]}`, so a crash report or a `dbg/1` can't print it.

## 15. Logging

Use `Logger` ([Logger](https://hexdocs.pm/logger/Logger.html)).

- **Levels**: `:debug` for detail useful while developing, `:info` for things that happened (a Job moved, the server started), `:warning` for something degraded that still works (an outbox row failing, checks failing), `:error` for an operation that failed and needs a person.
- **Set metadata once per process.** `Logger.metadata(job: number)` in a Job's `init/1` tags every line that process logs. Metadata keys are `snake_case` atoms, the same everywhere: `job`, `repo`, `stage`, `attempt`, `outbox_row` (provisional, section 19).
- **Pass a function for expensive messages**, `Logger.debug(fn -> ... end)`, so it's only built when the level is enabled.
- **Never log tokens, Ticket bodies or Stage Output bodies.** A Stage Output's body is the Operator's writing and can be long; log its document ID instead.
- **Don't log and return the same error.** Log where the error is handled, not at every layer it passes through.
- **Configure `:logger` handlers, not backends.** Backends moved out of Elixir into `logger_backends`, and 1.20 deprecates the `Logger.*_backend` functions in favour of handlers ([1.20 changelog](https://hexdocs.pm/elixir/changelog.html)).

## 16. Testing

ExUnit is the test framework. Test files end in `_test.exs` and mirror `lib/`; test-only modules go in `test/support/`, compiled only in `:test` through `elixirc_paths` ([`mix test`](https://hexdocs.pm/mix/Mix.Tasks.Test.html)).

### Pure code first

Most of Taiichi's rules live in `Jobs.Pipeline` and the parsing functions, so most tests need no database and no process. Build schema structs by hand and assert on what comes back:

```elixir
defmodule Taiichi.Jobs.PipelineTest do
  use ExUnit.Case, async: true

  alias Taiichi.Jobs.{Attempt, Pipeline}

  doctest Pipeline

  describe "next_attempt/4" do
    test "a failing Evaluation starts the next Build Attempt" do
      attempts = [
        %Attempt{seq: 1, stage: :plan, number: 1},
        %Attempt{seq: 2, stage: :build, number: 1},
        %Attempt{seq: 3, stage: :evaluate, number: 1}
      ]

      now = ~U[2026-10-22 12:00:00Z]

      assert %Attempt{stage: :build, number: 2, seq: 4} =
               Pipeline.next_attempt(List.last(attempts), attempts, :fail, now)
    end
  end
end
```

- **`async: true`** on every module that changes no global state. Tests in one module still run one at a time; modules run alongside each other ([`ExUnit.Case`](https://hexdocs.pm/ex_unit/ExUnit.Case.html)).
- **Assert with a pattern.** `assert %Attempt{stage: :build} = result` says what matters and ignores the rest, and its failure message shows the whole value.
- **`describe` per function or behaviour**, with test names that state the rule in domain language. `describe` blocks can't nest; use named setups instead.
- **Doctests** for small pure functions whose examples help the reader. They keep `@doc` examples from going stale.
- **`@tag :tmp_dir`** gives a test its own empty folder, unique per test so async tests don't collide. Use it for worktrees and Git repositories in tests.

### The database

`ecto_sqlite3` doesn't support async tests with `Ecto.Adapters.SQL.Sandbox`, because SQLite allows one write transaction at a time and the sandbox wraps each test in one ([`ecto_sqlite3`](https://hexdocs.pm/ecto_sqlite3/Ecto.Adapters.SQLite3.html#module-async-sandbox-testing)).

- **Tests that touch the database are `async: false`** and check out a sandbox connection in `setup`.
- **Use shared mode when other processes write.** A Job's process, its checker and the outbox sender run outside the test process; `Ecto.Adapters.SQL.Sandbox.mode(Taiichi.Repo, {:shared, self()})` lets them use the test's connection.
- **Keep the database tests few and the pure tests many.** Every rule that can be tested as a pure function should be.
- **Point `XDG_DATA_HOME` at a temporary folder** for the test run, so tests never touch the real database or worktrees.

### Processes

- **Start processes with `start_supervised!/2`.** ExUnit's test supervisor stops them before the next test starts ([`ExUnit.Callbacks`](https://hexdocs.pm/ex_unit/ExUnit.Callbacks.html)).
- **Never `Process.sleep/1` to wait for something.** Have the code under test send a message, or pass it a function that does, and `assert_receive` it. Give intervals as options (the checker's `:interval`) so tests use milliseconds.
- **`capture_log/1`** or `@tag :capture_log` for tests that log on purpose.

### Fakes for Forgejo

Mocks follow José Valim's rule: a mock is a noun, a module that implements an explicit contract, never a verb that patches an existing module at runtime ([Mocks and explicit contracts](https://dashbit.co/blog/mocks-and-explicit-contracts)). Patching modules globally stops tests from running concurrently and couples them to one implementation.

- **Behaviours get Mox mocks** (provisional). `Mox.defmock(Taiichi.TrackerMock, for: Taiichi.Tracker)` in `test_helper.exs`, the map of implementations in `config/test.exs` pointing at it, and `setup :verify_on_exit!` in each test module. A Job's process calls the mock from another process; allow it with `Mox.allow/3`, or use `set_mox_from_context` in non-async tests ([Mox](https://hexdocs.pm/mox/Mox.html)).
- **The Forgejo modules themselves are tested with `Req.Test`.** Point Req at a stub with `plug: {Req.Test, Taiichi.Tracker.Forgejo}` from test config, then stub responses per test with `Req.Test.stub/2` and `Req.Test.json/2`, or simulate a network failure with `Req.Test.transport_error/2` ([`Req.Test`](https://hexdocs.pm/req/Req.Test.html)). This checks the paths, statuses and JSON handling a mock of the behaviour would skip.

  ```elixir
  test "an unreachable Forgejo is forgejo_failed" do
    Req.Test.stub(Taiichi.Tracker.Forgejo, &Req.Test.transport_error(&1, :econnrefused))
    assert {:error, %Taiichi.Error{code: :forgejo_failed}} = Forgejo.check(@job)
  end
  ```

- **Pair the fakes with a real test.** José's article warns that a suite of only mocks is less reliable. The end-to-end smoke test against the host's Forgejo is that test.

### The router

`Plug.Test.conn/3` builds a request and `Router.call/2` runs it, with no socket ([`Plug.Test`](https://hexdocs.pm/plug/Plug.Test.html)). Assert on `conn.status` and on the decoded body, including the error body's `code`. Keep one test that talks over the real socket, with `Req.post!(url, unix_socket: path, json: body)`, to cover Bandit's configuration.

## 17. Documentation and comments

### `@moduledoc` and `@doc`

The Elixir team's guide draws the line: documentation is a contract with whoever calls the code; comments are for whoever reads the source ([writing documentation](https://hexdocs.pm/elixir/writing-documentation.html)).

- **Every public module has a `@moduledoc`** whose first line says what it is in domain terms. Tools show that line as the summary. Internal modules, such as the router's helpers, get `@moduledoc false`.
- **`@doc` on the public functions of public modules**, placed before the first clause. Start with a one-line summary; add `## Examples` with doctests where they help.
- **Refer to code in backticks with full names**, `Taiichi.Jobs.fetch/1`, `t:Taiichi.Error.t/0`, `c:Taiichi.Tracker.check/1`, so ExDoc and editors link them.
- **Don't document what the name says.** `@doc "Fetches a Job."` on `fetch/1` is noise; say what it returns when the Job doesn't exist instead.
- **No `@doc` on private functions.** The compiler discards it and warns.

### Code comments

A comment says what the code does and, where the names and types don't show it, the compressed reason. The bar is high: would the next editor break something they can't see from the code? If not, the comment shouldn't exist ([comments overuse](https://hexdocs.pm/elixir/code-anti-patterns.html#comments-overuse)).

- **Never reference a document outside the code.** No `docs/adr/…`, no spec section, no ticket number. The citation rots when the document is renumbered or superseded, and the reasoning belongs either in the comment, compressed, or in `docs/adr/`, where readers look for it. References run one way: an ADR or spec may point at code; code never points back.
- **Private functions almost never get a comment.** Name them for their reader. A comment survives only for a gotcha the code can't show: an ordering dependency, a deliberate absence, a trap.
- **If a comment translates a name, rename instead.** A comment that explains a unit (`# milliseconds`), a default, or what a function returns means the name is wrong: `@poll_interval_ms`, not `@poll_interval # ms`.
- **An ordering comment states the dependency and its reason, and stops**: `# The Checker reports to the Server, so the Server starts first.`
- **Name magic numbers** with module attributes rather than explaining them in comments.
- **Wrap comment prose at 80 columns.** `mix format` reflows code to 98 but leaves comment text alone, and 80 keeps comments readable in side-by-side diffs.

## 18. Conventions to adopt for Taiichi

The locally decided defaults. Diverge only with a comment saying why.

1. **The Mix app lives in `server/`; every module starts with `Taiichi.`**
2. **`mix format` on save; `--warnings-as-errors` for compile and test in CI.** The type checker's findings are warnings, so this is what makes them gate a change.
3. **Glossary terms in every name.** Words under a term's _Avoid_ list stay out of modules, functions, fields and atoms.
4. **Ecto schemas are the domain model; only each aggregate's store code calls `Taiichi.Repo`.** `Jobs.Pipeline` is pure functions over schema structs that return them unsaved.
5. **One public module per aggregate**, `Taiichi.Repos` and `Taiichi.Jobs`. The API calls only those.
6. **Atoms for closed sets, never built from input.** `Ecto.Enum` for columns; function clauses to parse strings.
7. **`{:error, %Taiichi.Error{}}` for every failure the API reports** (provisional shape); raise for programmer errors; no `try/rescue` for expected failures.
8. **A constraint violation becomes a domain error code**, through `unique_constraint/3` and Ecto's default index names. Never a raw database error at the API.
9. **One process per active Job is the only writer of that Job.** Reads go straight to the database. All of a process's messages are defined in its own module.
10. **Processes named through `Taiichi.Jobs.Registry`**, never through atoms.
11. **Explicit timeouts on every `GenServer.call/3` that waits on Git or Forgejo.**
12. **The checker's timer is `Process.send_after/3`**, rescheduled after each check. No Oban.
13. **The outbox is hand-rolled**: a table written in the same transaction as the save, and one sender process.
14. **One save per action**, through `Repo.transact/2`, with `default_transaction_mode: :immediate`. No Git or Forgejo call inside a transaction.
15. **The application's children are the startup order.** Migrations before the Job processes, the socket last.
16. **Responses built by hand in the API layer**, never by deriving `JSON.Encoder` on schemas.
17. **Implementations of a behaviour chosen from the Job's settings `type`**, through a map of literal module names.
18. **Git through `System.cmd/3` with an argument list and `GIT_TERMINAL_PROMPT=0`**, checking the exit status every time.
19. **Taiichi's own config files are read by our code with strict keys.** `config/` holds wiring only. Structs with tokens derive `Inspect` without them.
20. **Pure tests async; database tests `async: false` with the sandbox.** `start_supervised!/2` for processes, `assert_receive` instead of sleeping.
21. **Comments never reference documents outside the code.** Public modules have a `@moduledoc`; comment prose wraps at 80 columns.

## 19. Open questions

Style points where the community is split or this repo hasn't decided. Decide, record the answer in section 18, and delete the entry.

1. **Credo, and Styler.** Credo is the established linter; 1.7.19 supports Elixir 1.20 ([Credo](https://hexdocs.pm/credo/overview.html), [changelog](https://github.com/rrrene/credo/blob/master/CHANGELOG.md)). Its strict mode overlaps little with the type checker: it flags readability and design smells, not type errors. Styler is a formatter plugin that rewrites code toward one style automatically, and it has no per-rule configuration; Quokka is a configurable fork ([Styler](https://hexdocs.pm/styler/readme.html)). Recommendation: adopt Credo with `--strict` in CI, and leave Styler out until a style argument recurs, since it rewrites code silently.
2. **Dialyzer.** The compiler's type checker finds certain failures without annotations; Dialyzer checks typespecs, which the compiler ignores, and is slow and often noisy. The community is split in 2026 ([Elixir Forum](https://elixirforum.com/t/are-you-using-dialyzer-in-2026/75342)). Recommendation: no Dialyzer for now. Revisit when Elixir ships type signatures.
3. **How many `@spec`s.** Section 8's policy is provisional. The alternatives are specs everywhere (more to maintain, expected to be replaced) or specs only on callbacks.
4. **`JSON` or Jason.** The standard library's `JSON` (since 1.18) covers what the API needs and is what this guide uses. Jason will be in the dependency tree anyway (Credo requires it, for one), and `ecto_sqlite3` encodes `:map` columns with Jason unless `config :ecto_sqlite3, json_library: JSON` says otherwise. Recommendation: `JSON` in our code, and configure the adapter to match; confirm the adapter works with it before relying on it.
5. **The Forgejo client.** Req (on top of Finch and Mint) has the simplest API and `Req.Test` for stubbing. Finch directly would mean writing the JSON, auth and test stubbing ourselves. Recommendation: Req.
6. **The TOML library.** `toml` (by bitwalker, TOML 1.0, tested against the toml-test suite) and `toml_elixir` (TOML 1.0 and 1.1) are the candidates ([toml](https://hexdocs.pm/toml/readme.html), [toml_elixir](https://hexdocs.pm/toml_elixir/readme.html)). Either returns a map; strict keys are ours. Decide with #35, which first reads a Repo's config.
7. **Mox or hand-written fakes.** Mox verifies calls and works with async tests; a hand-written in-memory Tracker is simpler to read and can hold state across calls (labels added, then read back). José's article allows both, as long as the fake implements the behaviour. Recommendation: Mox for "this was called with that", an in-memory fake if a test needs a Tracker that remembers.
8. **Where Repo calls live.** The spec says only each module's "store code" calls `Repo`. This guide reads that as one `Store` module per aggregate (`Taiichi.Jobs.Store`). Alternatives: queries in each schema module, or private functions in `Taiichi.Jobs` itself.
9. **The error struct's shape.** Section 7's `Taiichi.Error` is one exception for every domain failure. Alternatives are bare atoms (`{:error, :job_moved}`), with messages built by the API, or one exception per error code. One struct keeps the message next to the knowledge that produced it; atoms keep the domain free of English.
10. **Logger metadata and format.** The keys in section 15 and whether the server logs plain text or a structured format are undecided. The Operator reads the log in a terminal today.
11. **Bounding a Git command.** `System.cmd/3` has no timeout. Running it in a `Task` and giving up after a while frees the Job's process but may leave the `git` process running. A `Port` gives more control at more code. Decide before Ship (#62), whose push is the likeliest to hang.
12. **The per-Job supervisor.** Section 9 shows the Job's process and checker under a small per-Job `:rest_for_one` supervisor. The alternative is both directly under the `DynamicSupervisor`, with the Job's process starting and stopping its checker. Decide with #42, which first starts a Job's process.

---

## Appendix A: Quick reference

```elixir
defmodule Taiichi.Jobs.Pipeline do
  @moduledoc """
  The Pipeline's rules, as pure functions over Job and Attempt structs.
  """

  alias Taiichi.Jobs.Attempt

  @doc """
  Returns the Stage a Job enters when a Stage Output is submitted.

      iex> Taiichi.Jobs.Pipeline.next_stage(:evaluate, :fail)
      :build
  """
  @spec next_stage(Attempt.stage(), :pass | :fail | nil) :: Attempt.stage()
  def next_stage(:plan, nil), do: :build
  def next_stage(:build, nil), do: :evaluate
  def next_stage(:evaluate, :pass), do: :ship
  def next_stage(:evaluate, :fail), do: :build

  @doc """
  Builds the Attempt that follows `current`, unsaved.
  """
  @spec next_attempt(Attempt.t(), [Attempt.t()], :pass | :fail | nil, DateTime.t()) ::
          Attempt.t()
  def next_attempt(%Attempt{} = current, attempts, verdict, now) do
    stage = next_stage(current.stage, verdict)

    %Attempt{
      job_id: current.job_id,
      seq: current.seq + 1,
      stage: stage,
      number: Enum.count(attempts, &(&1.stage == stage)) + 1,
      started_at: now
    }
  end
end
```

```elixir
# Store code: the one place that calls Taiichi.Repo.
def fetch(number) do
  case Repo.get(Job, number) do
    nil -> {:error, Error.new(:no_such_job, "No Job #{number}.")}
    job -> {:ok, job}
  end
end

# One save: an {:error, _} anywhere rolls back and comes out unchanged.
Repo.transact(fn ->
  with {:ok, job} <- insert_job(attrs),
       {:ok, _plan} <- insert_first_attempt(job) do
    {:ok, job}
  end
end)
```

```elixir
# A process: client functions and callbacks in one module, named via Registry.
def submit(number, submission),
  do: GenServer.call(via(number), {:submit, submission}, @submit_timeout)

defp via(number), do: {:via, Registry, {Taiichi.Jobs.Registry, number}}
```

```elixir
# Tests
use ExUnit.Case, async: true                         # pure code
use ExUnit.Case, async: false                        # touches the database
start_supervised!({Taiichi.Jobs.JobSupervisor, {7, interval: 10}})
assert_receive {:checked, 7}
assert {:error, %Taiichi.Error{code: :job_moved}} = Jobs.submit(7, submission)
```

## Appendix B: Reading list

- [Naming conventions](https://hexdocs.pm/elixir/naming-conventions.html) — the Elixir team's short page; read it once.
- The anti-pattern guides: [code](https://hexdocs.pm/elixir/code-anti-patterns.html), [design](https://hexdocs.pm/elixir/design-anti-patterns.html), [process](https://hexdocs.pm/elixir/process-anti-patterns.html), [macro](https://hexdocs.pm/elixir/macro-anti-patterns.html) — each entry has a problem, an example and a refactoring; half this guide comes from them.
- [Gradual set-theoretic types](https://hexdocs.pm/elixir/gradual-set-theoretic-types.html) and the [Elixir 1.20 release notes](https://elixir-lang.org/blog/2026/06/03/elixir-v1-20-0-released) — what the compiler checks and where it's going.
- [Writing documentation](https://hexdocs.pm/elixir/writing-documentation.html) — `@moduledoc`, `@doc`, doctests and linking.
- [`GenServer`](https://hexdocs.pm/elixir/GenServer.html), [`Supervisor`](https://hexdocs.pm/elixir/Supervisor.html), [`DynamicSupervisor`](https://hexdocs.pm/elixir/DynamicSupervisor.html), [`Registry`](https://hexdocs.pm/elixir/Registry.html) — the module docs are the best OTP reference there is.
- [To spawn, or not to spawn?](https://www.theerlangelist.com/article/spawn_or_not) by Saša Jurić — processes for runtime concerns, modules for thought concerns.
- [Good and bad Elixir](https://keathley.io/blog/good-and-bad-elixir.html) by Chris Keathley — opinions on `with`, pipes and errors from long production use.
- [Mocks and explicit contracts](https://dashbit.co/blog/mocks-and-explicit-contracts) by José Valim — why fakes implement behaviours.
- [The Elixir style guide](https://github.com/christopheradams/elixir_style_guide) — community conventions beyond what the formatter enforces.
- [`Ecto.Repo.transact/2`](https://hexdocs.pm/ecto/Ecto.Repo.html#c:transact/2), [`Ecto.Changeset`](https://hexdocs.pm/ecto/Ecto.Changeset.html), [`Ecto.Multi`](https://hexdocs.pm/ecto/Ecto.Multi.html) — transactions and constraints.
- [`Ecto.Adapters.SQLite3`](https://hexdocs.pm/ecto_sqlite3/Ecto.Adapters.SQLite3.html) — every pragma, and the limitations section.
- [`Plug.Router`](https://hexdocs.pm/plug/Plug.Router.html), [`Plug.Parsers`](https://hexdocs.pm/plug/Plug.Parsers.html), [Bandit](https://hexdocs.pm/bandit/Bandit.html) — the HTTP layer.
- [Req](https://hexdocs.pm/req/Req.html) and [`Req.Test`](https://hexdocs.pm/req/Req.Test.html) — the Forgejo client and its stubs.
- [`ExUnit.Case`](https://hexdocs.pm/ex_unit/ExUnit.Case.html) and [Mox](https://hexdocs.pm/mox/Mox.html) — tags, async, and concurrent mocks.
