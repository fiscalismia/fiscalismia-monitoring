# Plan: Wire config into main.go and probe a target

## Current State — What's broken and why


### 1. `config.go` — four bugs in `Load()`

#### Bug A: YAML is never unmarshalled

You read the file into `data` but never call `yaml.Unmarshal`.
After `os.ReadFile` you need:

```go
var config Config
if err := yaml.Unmarshal(data, &config); err != nil {
    return nil, fmt.Errorf("unmarshal config: %w", err)
}
```

**Go lesson — `yaml.Unmarshal`:**
`yaml.Unmarshal([]byte, &target)` is Go's equivalent of Python's `yaml.safe_load()`.
The second arg must be a *pointer* so the library can write into it.
Struct tags (`yaml:"name"`) control the mapping — same concept as Pydantic's `Field(alias=...)`.

#### Bug B: `config` variable is referenced but never declared

The for-loop uses `config.Targets` — but `config` doesn't exist in scope yet.
This is what Bug A's fix provides. After unmarshalling, `config` is your local variable.

#### Bug C: comparing `string` to `0`

`Target.Timeout` is typed `string` but the loop compares it to `0` (an int).
Go is strictly typed — this won't compile. Two options:

- **Option 1 (simple):** keep `Timeout` as `string`, compare with `""` (empty string),
  and parse to `time.Duration` when you actually need it.
- **Option 2 (cleaner):** change `Timeout` to `time.Duration`. The `yaml.v3` library
  can unmarshal `"3s"` directly into `time.Duration`. Then comparing to `0` works.

Recommendation: use `time.Duration` for both `GlobalTimeout` and `Timeout`.

#### Bug D: function never returns

`Load()` is declared as returning `(*Config, error)` but has no `return` statement.
After the timeout-defaulting loop, add:

```go
return &config, nil
```

**Go lesson — naked returns vs explicit returns:**
Go supports named return values (`func Load() (cfg *Config, err error)`) with a bare
`return`, but the community consensus is: *always return explicitly*. It's clearer.

---

### 2. `Config` struct doesn't match YAML shape

Your YAML has *nested maps* under `targets`:

```yaml
targets:
  internal:
    - name: "Demo Frontend"
      ...
  external:
    - name: "Demo Frontend"
      ...
```

But your struct says:

```go
Targets []Target `yaml:"targets"`
```

That expects `targets` to be a flat list. It won't unmarshal the nested map.

**Fix:** change the struct to match the YAML shape:

```go
type TargetGroups struct {
    Internal []Target `yaml:"internal"`
    External []Target `yaml:"external"`
}

type Config struct {
    GlobalTimeout time.Duration `yaml:"global_timeout"`
    Targets       TargetGroups  `yaml:"targets"`
}
```

Then to get all targets as one slice you can add a helper method:

```go
func (tg TargetGroups) All() []Target {
    return append(tg.Internal, tg.External...)
}
```

**Go lesson — methods on structs:**
`func (tg TargetGroups) All()` is a *value receiver* method.
It's Go's equivalent of `def all(self)` in Python or a class method in TS.
Use a *pointer receiver* `func (tg *TargetGroups)` only when you need to mutate the struct.

---

### 4. `main.go` — loading config and probing one HTTP target

Here's the wiring you need, step by step:

#### Step 1: call `config.Load()`

```go
cfg, err := config.Load("configs/targets.yml")
if err != nil {
    log.Fatal(err)   // log.Fatal prints and calls os.Exit(1)
}
```

**Go lesson — `log.Fatal` vs `panic`:**
`log.Fatal` is for *expected* startup failures (can't read config, can't bind port).
`panic` is for programmer errors / impossible states. Never `panic` for I/O errors.

#### Step 2: pick a target (hardcoded for now)

Find the external backend target. Simplest approach — just loop:

```go
var target config.Target
for _, t := range cfg.Targets.External {
    if t.Name == "Demo Backend" {
        target = t
        break
    }
}
```

**Go lesson — no `filter()` / `find()`:**
Go deliberately has no `Array.find()` or list comprehension. A plain for-loop is idiomatic.
Generics (`slices.IndexFunc` from Go 1.21+) exist but a loop is clearer for learning.

#### Step 3: make an HTTPS request

```go
client := &http.Client{
    Timeout: cfg.GlobalTimeout,
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    },
}
resp, err := client.Get(target.URL)
if err != nil {
    log.Fatalf("probe %s failed: %v", target.Name, err)
}
defer resp.Body.Close()     // ← ALWAYS defer Close on HTTP responses
```

**Go lesson — `defer`:**
`defer resp.Body.Close()` schedules the close for when the function returns.
It's Go's version of Python's `with` / TS's `using`. Without it you leak connections.
The `bodyclose` linter in your `.golangci.yml` catches this — that's why it's there.

**Go lesson — `InsecureSkipVerify`:**
Your `.golangci.yml` already excludes `G402` from gosec for this exact reason:
internal probes hit IPs without valid certs. For the external probe to
`backend.demo.fiscalismia.com` you technically don't need it. Consider making
`InsecureSkipVerify` configurable per-target or per-group later.

#### Step 4: read and print the response

```go
body, err := io.ReadAll(resp.Body)
if err != nil {
    log.Fatalf("read body: %v", err)
}
fmt.Printf("[%d] %s → %s\n", resp.StatusCode, target.Name, string(body))
```

**Go lesson — `io.ReadAll`:**
Reads the entire body into `[]byte`. Fine for health check responses (small).
For large responses you'd use `io.LimitReader` to cap memory usage.

#### Imports you'll need in `main.go`:

```go
import (
    "crypto/tls"
    "fmt"
    "io"
    "log"
    "net/http"

    "github.com/fiscalismia/fiscalismia-monitoring/internal/config"
)
```

**Go lesson — import paths:**
The last segment (`config`) becomes the package name you use in code: `config.Load(...)`.
The full path is your module path from `go.mod` + the directory path. This is how Go
resolves internal packages — no `__init__.py`, no `index.ts`, just the directory.

---

## Summary checklist

| # | File | What to do |
|---|------|-----------|
| 1 | `go.mod` | Change `go 1.26` → `go 1.24`, run `go mod tidy` |
| 2 | `config.go` | Add `yaml.Unmarshal(data, &config)` call |
| 3 | `config.go` | Change `Targets []Target` → `Targets TargetGroups` (nested struct matching YAML) |
| 4 | `config.go` | Change `Target.Timeout` from `string` to `time.Duration` (or compare to `""`) |
| 5 | `config.go` | Add `return &config, nil` at end of `Load()` |
| 6 | `main.go` | Import `internal/config`, call `config.Load()` |
| 7 | `main.go` | Create `http.Client` with timeout, `GET` the target URL, print status + body |

## Build & run after changes

```bash
go mod tidy          # sync dependencies
go vet ./...         # catch issues the compiler misses
go build ./cmd/healthcheck/
./healthcheck        # should print the health check response
```

**Go lesson — `go vet`:**
Run it before every commit. It's a lightweight static analyzer built into the toolchain.
Think of it as Go's equivalent of `mypy` or `tsc --noEmit`. Your `golangci-lint run`
already includes `govet`, but `go vet` is faster for quick checks.
