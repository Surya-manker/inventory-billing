# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Layout

Two sibling projects share this directory:

| Directory | Stack | Port |
|---|---|---|
| `inventory-billing/` | Go 1.24 · Gin · GORM · MySQL · Redis · Asynq | `8080` |
| `inventory-billing-ui/` | React 18 · Vite · Tailwind CSS · Axios | `5173` |

---

## Backend (`inventory-billing/`)

### Commands

All commands run from inside `inventory-billing/`.

```bash
# Run the API server
go run ./cmd/server/main.go

# Run the background worker (Asynq + scheduled jobs) — must run alongside the server
go run ./cmd/worker/main.go

# Seed the database with sample data
go run ./cmd/seed/main.go

# Build all binaries
go build ./...

# Format
gofmt -w .

# Tidy dependencies
go mod tidy
```

No test suite exists yet.

### Environment

Copy `app.env.example` → `app.env` before first run. Three keys are required (the process fails fast if missing):

```
DATABASE_DSN=root:@tcp(127.0.0.1:3306)/inventory_billing?charset=utf8mb4&parseTime=True&loc=Local
JWT_SECRET=<random string, min 32 chars>
REDIS_ADDR=localhost:6379
```

Optional but needed for full functionality:

```
GST_SELLER_GSTIN=<15-char GSTIN>       # required for GST invoice generation
GST_STATE_CODE=<first 2 digits of GSTIN>
MAIL_PROVIDER=smtp                      # default "noop" discards all email
STORAGE_PROVIDER=local                  # default; PDFs saved to ./data/assets/
WORKER_CONCURRENCY=10                   # Asynq goroutines in the worker process
```

### Three-process Architecture

The system runs as three separate OS processes that share the same MySQL and Redis:

```
cmd/server/main.go   ← HTTP API (Gin)         — port 8080
cmd/worker/main.go   ← Background jobs         — no port
cmd/seed/main.go     ← One-shot DB seeder      — exits after run
```

The worker runs two concurrent systems:
1. **Asynq server** — processes tasks enqueued by the API (PDF generation → email)
2. **Scheduler** (`pkg/jobs/runner.go`) — fires recurring jobs on a timer (low-stock alert hourly, overdue invoices daily, daily snapshot daily)

### Request / Job Flows

**HTTP request flow:**
```
HTTP → Middleware chain → Handler → Service → Repository → GORM → MySQL
```

Middleware chain (in order): `gin.Recovery → Logger → RateLimiter(20 rps, burst 50) → AuditLog`

**Async job flow (invoice PDF example):**
```
POST /invoices  →  invoiceService.Create() [sync, DB tx]
                →  queue.Client.EnqueueInvoicePDF() [non-blocking, writes to Redis]
                →  returns 201 with pdf_job_id

Worker process:
  InvoicePDFHandler.ProcessTask()
    → updates job_records status=processing
    → invoiceRepo.FindByID()
    → pdf.Generator.Generate()
    → storage.Put() [saves to ./data/assets/invoices/…]
    → updates job_records status=completed, result={pdf_url}
    → queue.Client.EnqueueEmail() [enqueues follow-up]
  EmailHandler.ProcessTask()
    → storage.Get() [fetches PDF bytes]
    → mailer.Send() [SMTP with PDF attached]
```

Client polls `GET /api/v1/jobs/:id` (202 while processing, 200 when done) or `GET /api/v1/invoices/:id/pdf` (same semantics, filtered to PDF jobs for that invoice).

### Code Architecture

```
cmd/
  server/    HTTP API entry point — wires config → DB → router.Setup()
  worker/    Background processor — wires Asynq server + jobs.Runner
  seed/      Seed script

config/config.go          Single Config struct; Load() reads app.env via Viper

internal/
  domain/    Pure structs + domain errors. No DB or HTTP imports.
             errors.go — all sentinel errors (domain.ErrNotFound, etc.)
             job.go    — JobRecord + JobStatus constants
             One file per aggregate: category, credit_note, customer, invoice,
             invoice_counter, payment, product, stock_log, user, vendor, warehouse.
  handler/   HTTP layer: parse request → call service → write response. One file per
             aggregate (auth, category, credit_note, customer, invoice, job, payment,
             product, report, stock, user, vendor).
             invoice_handler.go — also owns GET /invoices/:id/pdf + PDF job enqueue
             job_handler.go     — GET /jobs/:id
  service/   Business logic. One file per aggregate. invoice_service.go is the most
             complex (atomic tx).
  repository/ GORM queries, one file per aggregate root.
             job_repository.go — CRUD + UpdateJobParams for job tracking
  middleware/ auth.go, rate_limiter.go, logger.go, audit.go
  router/router.go   Wires every dependency; all routes defined here.

pkg/
  cache/     redis.go — TokenStore (refresh tokens). generic.go — GetOrSet[T] helper.
  database/  Connect() + AutoMigrate (runs on every startup in dependency order).
  jobs/      runner.go — simple ticker-based scheduler. Three job implementations.
  jwt/       JWT sign/verify helpers.
  logger/    Zap logger setup (go.uber.org/zap).
  mailer/    Mailer interface. SMTPMailer (gopkg.in/gomail.v2). NoopMailer (dev).
  pdf/       Generator interface. fpdf implementation (go-pdf/fpdf). GST invoice layout.
  queue/     Asynq wrapper.
             tasks.go              — task type constants + typed payload structs
             client.go             — typed enqueue methods; creates job_records on enqueue
             server.go             — Asynq server setup + logging middleware
             invoice_pdf_handler.go — PDF worker handler (implements asynq.Handler)
             email_handler.go      — email worker handler
  storage/   Storage interface. LocalStorage (filesystem). S3Storage stub.
  utils/     context.go, response.go, gst.go, math.go, password.go, query.go
```

### Key Conventions

**Identity in handlers:** Always use `utils.UserIDFromCtx(c)` / `utils.UserRoleFromCtx(c)` — never read `user_id` from the request body.

**Authorization:** `p` group = any valid Bearer token. `admin` group = Bearer + `RequireRole("admin")`. Both are defined in `router.go` and share the same JWT secret.

**Responses:** Always use `utils.SuccessResponse` / `utils.ErrorResponse` / `utils.PaginatedResponse`. Never write `c.JSON` directly in handlers.

**Domain errors:** Define in `internal/domain/errors.go`. Map to HTTP status codes in the handler's `switch` block, not in the service.

**Cache invalidation:** `productService.invalidateProductCache()` must use `cache.DelByPrefix` (SCAN + DEL), not `cache.Del`, because `redis.Del` does not expand wildcards.

**Asynq retry behaviour:** Return `fmt.Errorf("%w: ...", asynq.SkipRetry)` for permanent failures (bad payload, entity deleted). Return a plain error for transient failures (DB timeout, SMTP error) — Asynq will retry with exponential back-off (`delay = n² seconds`).

**Job tracking:** `queue.Client.enqueue()` always creates a `job_records` row after a successful Asynq enqueue. The DB write is best-effort (logged on failure, never blocks the caller). Handlers update the row's status at start, completion, and on each failure.

### Critical Subsystems

**Invoice creation** (`internal/repository/invoice_repository.go → CreateTx`): Single GORM transaction that locks product rows with `SELECT … FOR UPDATE` in sorted UUID order (prevents AB/BA deadlock), re-validates stock, generates the invoice number via `InvoiceCounter` auto-increment, inserts invoice + items, deducts stock, and appends stock_log entries.

**GST logic** (`pkg/utils/gst.go`): `intra_state` is true when the first 2 digits of the customer's GSTIN match `GST_STATE_CODE` from config. Intra-state → CGST + SGST split equally; inter-state → IGST.

**Product caching** (`internal/service/product_service.go`): List responses are cached in Redis for 3 minutes with a deterministic key built from all filter + pagination parameters. Cache is invalidated on create, update, delete, and stock adjust by calling `invalidateProductCache()`.

**Daily snapshot** (`pkg/jobs/daily_snapshot.go`): Runs nightly. Writes aggregated stats to Redis at `snapshot:daily:YYYY-MM-DD`. The report handlers do not yet read from this cache — they query the live DB.

**Schema management:** GORM `AutoMigrate` runs on every `database.Connect()` call. Models must be listed in dependency order in `pkg/database/database.go`. The `migrations/` folder contains reference SQL only — it is not executed automatically.

### Health Endpoints

```
GET /health        — always 200 (liveness)
GET /health/ready  — 200 if DB is reachable, 503 otherwise (readiness)
```

### Static File Serving

When `STORAGE_PROVIDER=local`, the API server serves generated PDFs at `/assets/*` mapped to `STORAGE_LOCAL_DIR` (`./data/assets` by default). Generated PDFs are stored at `invoices/<invoice_number>/<invoice_id>.pdf` within that directory.

---

## Frontend (`inventory-billing-ui/`)

### Commands

```bash
npm install        # first time only
npm run dev        # dev server at http://localhost:5173
npm run build      # production build → dist/
npm run preview    # preview production build locally
```

No test suite or linter is configured.

### Architecture

```
src/
  api/axios.js       Axios instance; baseURL=/api/v1; Bearer token injected by interceptor;
                     401 → clears localStorage + redirects to /login.
  api/*.js           One thin file per resource. No logic, just axios calls.
  context/AuthContext  { user, login, logout, isAdmin }. login() persists tokens to localStorage.
  components/layout/   Layout.jsx = protected route wrapper. Sidebar.jsx = nav + logout.
  components/common/   Stateless: Modal, ConfirmDialog, Badge, Pagination, LoadingSpinner.
  pages/             One file per route; owns its own data-fetching, loading, and error state.
  utils/formatters.js  formatCurrency (INR), formatDate, formatDateTime, statusColors.
```

**Routing:** React Router v6. `/login` and `/register` are public. Everything else is nested under `<Layout>` (enforces auth). `*` catch-all → `/dashboard`.

**Dev proxy:** Vite proxies `/api/*` → `http://localhost:8080`. The axios `baseURL` is `/api/v1` (relative). No CORS config needed in dev. Production requires a reverse proxy forwarding `/api` to Go.

**Role gating:** `useAuth().isAdmin` (`user.role === 'admin'`) gates Add/Edit/Delete buttons in page components. Staff can view everything and create invoices.

**Tailwind component classes** — defined as `@layer components` in `src/index.css`:
`.btn`, `.btn-primary`, `.btn-secondary`, `.btn-danger`, `.btn-sm`, `.input`, `.label`, `.card`, `.table-th`, `.table-td`

---

## Running Everything Together

```bash
# Terminal 1 — API server
cd inventory-billing && go run ./cmd/server/main.go

# Terminal 2 — background worker
cd inventory-billing && go run ./cmd/worker/main.go

# Terminal 3 — frontend
cd inventory-billing-ui && npm run dev
```

Open `http://localhost:5173`. First visit → `/register` to create an admin account.
