# Go Standards

**A reference implementation of the Go service conventions I hold teams to.** Every file here demonstrates a standard, and the service they build runs for real: a JSON API over PostgreSQL with unit and integration tests, a mutation gate, and a container image.

Its companions, [cypress-standards](https://github.com/craig-hunt/cypress-standards), [playwright-standards](https://github.com/craig-hunt/playwright-standards), and [playwright-standards-csharp](https://github.com/craig-hunt/playwright-standards-csharp), apply the same discipline to browser test suites. This project serves the domain those suites drive in a browser, a task list, a signup form, and an inventory table, as an API.

**Platform.** These conventions assume Linux or WSL with Docker, where Go services build and run. Commands appear in bash form throughout, unlike the Windows-first companions.

---

## The standards

### 1. No magic strings or magic numbers, anywhere, tests included

Every literal carrying meaning beyond its immediate expression lives in a named constant: environment variable names, routes, SQL, log messages, error codes, user-facing copy, timeouts, and test data.

```go
// Wrong
if len(title) > 200 {
mux.HandleFunc("GET /api/tasks", h.list)

// Right
if utf8.RuneCountInString(trimmed) > MaxTitleLength {
mux.HandleFunc(RouteList, h.list)
```

Each package keeps its values in `constants.go`, and its tests keep theirs in `constants_test.go`. Tests reference the production constants, so a renamed route or a reworded message fails once, at compile time, instead of drifting silently.

**Three layers enforce it.** `conventions_test.go` parses every Go file and fails on any literal outside a constant declaration, a struct tag, a constants file, or a test case name. golangci-lint adds `mnd` for numbers and `goconst` for repeated strings. Review covers what remains.

**Where Go differs.** The Cypress project enforces this standard with ESLint rules on three call shapes. Go has no such rule, so this project walks the syntax tree in a test, which reaches every literal rather than a few call sites.

### 2. Twelve-factor processes

The service follows [The Twelve-Factor App](https://12factor.net).

| Factor              | Where it shows                                                                                       |
| ------------------- | ---------------------------------------------------------------------------------------------------- |
| Codebase            | One repository, one module, two commands built from it                                               |
| Dependencies        | `go.mod` and `go.sum` declare and pin every dependency                                               |
| Config              | `internal/config` reads only the environment and refuses to start without `DATABASE_URL` or `API_TOKEN` |
| Backing services    | PostgreSQL attaches by URL and swaps without a code change                                            |
| Build, release, run | The Dockerfile builds one image; configuration arrives at run time                                   |
| Processes           | Stateless; every piece of state lives in PostgreSQL                                                  |
| Port binding        | The service serves its own HTTP on `PORT`                                                             |
| Concurrency         | Identical processes scale out behind a load balancer                                                  |
| Disposability       | `internal/server` stops accepting work on SIGTERM and drains requests within `SHUTDOWN_TIMEOUT`      |
| Dev/prod parity     | `compose.yaml` and the integration suite run the production PostgreSQL image                          |
| Logs                | One JSON event per line on stdout; the platform routes the stream                                    |
| Admin processes     | `cmd/migrate` applies migrations and seed data as a one-off run                                       |

**Environment files never live in the repository.** Go reads no `.env` file. Keep one outside the project and export it into the shell:

```bash
set -a; . "$HOME/.secrets/go-standards.env"; set +a
```

Ignoring a file stops it reaching a commit, not a reader. AI coding assistants, editor extensions, and language servers read anything inside the project. `.env.example` documents the variables and holds no values.

### 3. Components with boundaries a test enforces

- **Commands hold wiring only.** `cmd/demoapi` loads configuration, connects, assembles, and runs. Every behavior it assembles lives, and gets tested, in an `internal` package.
- **Interfaces live at the consumer.** Each domain package (`task`, `signup`, `inventory`) defines the `Store` it needs, and `internal/postgres` satisfies it. The domain never imports persistence, and `conventions_test.go` fails the build if a domain package imports `internal/postgres`, `internal/api`, or pgx.
- **Packages carry names for what they do.** The same test rejects `util`, `common`, `helpers`, `shared`, `misc`, and `base`. A package named for its role collects unrelated code; a package named `httpjson` or `requestid` keeps a boundary.

### 4. Reuse over repetition

Shared behavior lives once, in a package named for its job.

| Package                           | Reused by                                                                                  |
| --------------------------------- | ------------------------------------------------------------------------------------------ |
| `httpjson`                        | Every handler: JSON writes, uniform error bodies, and generic, size-limited request decoding |
| `middleware`                      | The whole API: request IDs, access logging, panic recovery, bearer authentication          |
| `logging`, `config`               | Both commands                                                                              |
| `expect`, `apitest`, `logcapture` | Every test package                                                                         |
| `task/storetest`                  | The in-memory fake and the PostgreSQL store, which run one shared contract                 |

**The store contract earns its place.** Handler tests run against an in-memory store, and they prove something only while that fake behaves like the database. `storetest.Run` holds one set of behaviors that both stores must pass, so the fake cannot drift from the real thing.

### 5. Types over primitives where a domain exists

`task.Title` comes only from `NewTitle`, which trims and length-checks it, so every title a store receives already passed validation. `Filter`, `Plan`, `Column`, `Direction`, and `Status` narrow values the compiler checks.

### 6. Errors return, wrap, and match by identity

- Functions return errors. Nothing panics except a programmer error, and `middleware.Recover` turns an escaped panic into a logged 500 rather than a dead process.
- Wrapping uses `%w`, and callers match with `errors.Is`. Nothing compares error text.
- Error text stays lowercase and unpunctuated, as Go convention expects. User-facing sentences live in separate `Msg` constants.
- `httpjson.WriteInternal` logs the cause with the request ID and returns a generic body, so internal detail never reaches a caller.

### 7. Context flows through every blocking call

Handlers pass the request context to the store, readiness bounds its database ping with `ReadyTimeout`, and shutdown derives its deadline from the run context. Every timeout carries a name.

### 8. Structured, correlated logs

`log/slog` writes JSON with an ISO 8601 UTC timestamp, level, service name, and message on every line. `middleware.RequestID` accepts or generates an `X-Request-ID`, and every line a request produces carries it. The linter forbids `fmt.Print`, `print`, and the `log` package.

### 9. Security at every boundary

- **Authentication on every API route.** `middleware.RequireBearerToken` guards all of `/api/`, compares tokens in constant time, and answers 401 with `WWW-Authenticate`. Only the health probes stay open, since a platform calls them before any credential exists.
- **Parameterized SQL only.** Queries live in constants with `$n` placeholders.
- **Validation at the edge.** Bodies decode under a size limit and reject unknown fields, and domain types validate before a store sees a value.
- **Scanning in CI.** `gosec` through golangci-lint, `govulncheck`, Gitleaks, and Dependabot.

### 10. Tests read as behavior statements

- Standard `testing` only, with shared helpers in `internal/expect`. No assertion library.
- Names describe behavior and outcome: `TestCreateNamesTheTitleWhenItIsBlank`, `TestReadinessReportsUnavailableWhenTheDatabaseFails`. Duplicate names count as a defect.
- Table cases carry a `name` that reads as a statement.
- Integration tests carry the `integration` build tag and run against a throwaway PostgreSQL container that `scripts/with-test-postgres.sh` creates and removes. Each test gets its own schema, so parallel runs never share rows.

**Where Go differs.** The TypeScript and C# companions lean on their runners' assertion APIs. Go's standard library supplies none, and a third-party assertion library adds a dependency to every test package, so `internal/expect` supplies four small generic helpers instead.

### 11. Mutation testing gates the build at 70%

[gremlins](https://github.com/go-gremlins/gremlins) mutates the code and reruns the suite, integration tests included. **The build fails below 70% efficacy (killed mutants over killed plus lived) or below 70% mutant coverage.** Efficacy alone counts only mutants some test reaches, so the coverage threshold stops untested code from slipping past the gate.

Two exclusions keep the gate honest rather than lenient. `cmd/` holds wiring only. Constants files hold compile-time values that tests reference by name, so a mutated constant changes the production value and the test's expectation together, and no test can ever see the difference.

gremlins exits with an error when either threshold misses, which makes it a gate rather than a report. It remains a 0.x release, so the Makefile pins its version.

### 12. One authority on formatting

gofmt and goimports own layout, and imports group as standard library, external, then this module. Nobody debates style in review.

### 13. Comments explain why, never what

A comment that restates the code adds a second thing to maintain. A comment recording a constraint, a rejected alternative, or a non-obvious consequence saves the next reader an investigation. `recoverPanic` carries one: `recover` stops a panic only when the deferred function calls it directly, which nothing in the code itself reveals.

### 14. Lint findings fail the build

golangci-lint runs its standard set plus `bodyclose`, `contextcheck`, `errorlint`, `forbidigo`, `goconst`, `gosec`, `mnd`, `noctx`, `rowserrcheck`, `sqlclosecheck`, and `usestdlibvars`. A warning that never fails teaches a team to scroll past it.

---

## Structure

```
cmd/demoapi/                    API server: configuration, wiring, graceful run
cmd/migrate/                    admin process: migrations and demo data
internal/config/                environment configuration
internal/logging/               JSON logger
internal/requestid/             per-request identifiers
internal/httpjson/              JSON bodies and error shapes
internal/middleware/            request ID, access log, recovery, authentication
internal/health/                liveness and readiness probes
internal/server/                HTTP server lifecycle
internal/api/                   routes and middleware, assembled
internal/task/                  tasks: domain, handler
internal/task/storetest/        the contract every task store passes
internal/signup/                signups: validation, handler
internal/inventory/             inventory: search, sort, handler
internal/postgres/              stores, migrations, seed data
internal/expect/                assertion helpers for tests
internal/apitest/               HTTP request helpers for tests
internal/logcapture/            log capture for tests
conventions_test.go             enforces standards 1 and 3
scripts/with-test-postgres.sh   throwaway database for integration and mutation runs
```

---

## API

Every `/api/` route requires `Authorization: Bearer <API_TOKEN>`.

| Method   | Path              | Behavior                                                             |
| -------- | ----------------- | -------------------------------------------------------------------- |
| `GET`    | `/health`         | Liveness; answers 200 without consulting the database                 |
| `GET`    | `/health/ready`   | Readiness; 503 when the database does not answer                     |
| `GET`    | `/api/tasks`      | Tasks with remaining and total counts; `?filter=all\|active\|completed` |
| `POST`   | `/api/tasks`      | Creates a task from `{"title": "..."}`                               |
| `PATCH`  | `/api/tasks/{id}` | Sets `{"completed": true\|false}`                                     |
| `DELETE` | `/api/tasks/{id}` | Deletes one task                                                     |
| `DELETE` | `/api/tasks`      | Clears completed tasks and reports how many                          |
| `POST`   | `/api/signups`    | Validates a signup, naming every field problem at once               |
| `GET`    | `/api/inventory`  | Items with `?search=`, `?sort=name\|quantity\|status`, `?direction=ascending\|descending` |

---

## Getting started

Requires Go and Docker.

```bash
export POSTGRES_PASSWORD="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
export API_TOKEN="$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')"
docker compose up --build
```

In a second shell with the same `API_TOKEN` exported:

```bash
curl -s localhost:8080/health/ready
curl -s -H "Authorization: Bearer $API_TOKEN" localhost:8080/api/tasks
```

Compose starts PostgreSQL, runs `migrate` once, and then starts the API.

---

## Verification

```bash
make verify         # everything below, as CI runs it
make format-check   # gofmt
make vet            # go vet, with and without the integration tag
make lint           # golangci-lint
make test           # unit tests with the race detector
make integration    # unit and integration tests against a throwaway PostgreSQL
make mutation       # gremlins, failing below 70%
make vulncheck      # govulncheck
make secrets        # Gitleaks
```

Tool versions pin in the Makefile and run through `go run`, so nothing needs installing beyond Go and Docker. The CI workflow pins each action to a commit SHA rather than a tag, so a moved tag cannot change what runs.

---

## Configuration

| Variable           | Default             | Purpose                                      |
| ------------------ | ------------------- | -------------------------------------------- |
| `DATABASE_URL`     | none, required      | PostgreSQL connection URL                    |
| `API_TOKEN`        | none, required      | Bearer token every `/api/` request presents  |
| `SERVICE_NAME`     | `go-standards-demo` | Service name on every log line               |
| `PORT`             | `8080`              | Listening port                               |
| `LOG_LEVEL`        | `info`              | `debug`, `info`, `warn`, or `error`          |
| `SHUTDOWN_TIMEOUT` | `10s`               | How long shutdown waits for in-flight requests |

---

## Adopting this

Take the constants discipline, the conventions test, consumer-side interfaces, and the Makefile gates. Replace the three feature packages with your own domain and `conventions_test.go`'s domain list with your packages.

Keep `storetest`. A fake that never runs the real store's contract proves only that it agrees with itself.

If you adopt these standards, retain the copyright notice per the MIT license.

---

_Standards authored by Craig Hunt. Licensed MIT._
