# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Layout

Two sibling projects share this directory:

| Directory | Stack | Port |
|---|---|---|
| `inventory-billing/` | Go 1.22 · Gin · GORM · MySQL · Redis | `8080` |
| `inventory-billing-ui/` | React 18 · Vite · Tailwind CSS · Axios | `5173` |

---

## Backend (`inventory-billing/`)

### Commands

```bash
# Run (development)
go run ./cmd/server/main.go

# Build binary
go build -o inventory-billing ./cmd/server

# Format
gofmt -w .

# Tidy dependencies
go mod tidy
```

No test suite exists yet.

### Environment

Copy `app.env.example` → `app.env` before first run. Required values:

```
DATABASE_DSN=root:@tcp(127.0.0.1:3306)/inventory_billing?charset=utf8mb4&parseTime=True&loc=Local
JWT_SECRET=<random string>
REDIS_ADDR=localhost:6379
GST_SELLER_GSTIN=<15-char GSTIN>
GST_STATE_CODE=<first 2 digits of GSTIN>
```

Config is loaded by `config/config.go` via Viper; all keys map directly to struct fields.

### Architecture

Request flow: `HTTP → Middleware → Handler → Service → Repository → GORM → MySQL`

```
cmd/server/main.go          Entry point; wires config → DB → Redis → router
config/config.go            Single Config struct; Load() reads app.env via Viper
internal/
  domain/                   Pure structs + domain errors. No DB or HTTP logic.
  handler/                  HTTP layer only: parse request, call service, write response.
  service/                  Business logic: GST calc, invoice numbering, stock mutations.
  repository/               GORM queries. One file per aggregate root.
  middleware/               auth.go (Bearer JWT → context), rate_limiter.go, logger.go
  router/router.go          Wires all dependencies; defines all routes under /api/v1/
pkg/
  database/database.go      Connect() + AutoMigrate (runs in dependency order)
  cache/redis.go            TokenStore: SaveRefreshToken / GetRefreshToken / DeleteRefreshToken
  utils/
    jwt.go                  GenerateAccessToken, ParseJWT
    context.go              UserIDFromCtx, UserRoleFromCtx  ← read user from Gin context here
    response.go             SuccessResponse / ErrorResponse  ← always use these for JSON replies
    gst.go                  GST tax calculation (CGST/SGST vs IGST)
```

**Key conventions:**
- User identity in handlers comes from `utils.UserIDFromCtx(c)` / `utils.UserRoleFromCtx(c)`, set by `middleware.Auth()`.
- Admin-only routes are wrapped with `middleware.RequireRole("admin")` in `router.go`.
- All JSON responses go through `utils.SuccessResponse` / `utils.ErrorResponse` — never write raw `c.JSON`.
- Domain errors (e.g. `domain.ErrEmailTaken`, `domain.ErrInsufficientStock`) are defined in `internal/domain/errors.go` and mapped to HTTP status codes in handlers.

**Invoice creation** is the most complex flow: `invoice_service.go` atomically creates invoice rows, deducts stock, appends stock_log entries, and generates an invoice number — all inside a single GORM transaction.

**GST logic:** `intra_state` is determined by comparing the first 2 digits of the customer's GSTIN with `GST_STATE_CODE` from config. Intra-state → CGST + SGST; inter-state → IGST. Calculation lives in `pkg/utils/gst.go`.

**Database schema** is managed entirely by GORM `AutoMigrate` on startup — no migration runner. The SQL files under `migrations/` are reference documents only and are not executed automatically.

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
  api/axios.js          Axios instance; baseURL=/api/v1; request interceptor adds Bearer token;
                        response interceptor clears localStorage + redirects on 401.
  api/*.js              One file per resource (auth, products, customers, invoices, dashboard).
                        Each exports thin functions — no logic, just axios calls.
  context/AuthContext   Single source of truth for auth state. Provides { user, login, logout, isAdmin }.
                        login() stores access_token, refresh_token, user in localStorage.
  components/layout/    Layout.jsx is the protected route wrapper (checks user, redirects to /login).
                        Sidebar.jsx renders nav + logout.
  components/common/    Modal, ConfirmDialog, Badge, Pagination, LoadingSpinner — all stateless.
  pages/                One file per route. Pages own their data-fetching, loading, and error state.
  utils/formatters.js   formatCurrency (INR), formatDate, formatDateTime, statusColors map.
```

**Routing:** React Router v6. `/login` and `/register` are public. All others are nested under `<Layout>` which enforces auth. The `*` catch-all redirects to `/dashboard`.

**Proxy:** Vite dev server proxies `/api/*` → `http://localhost:8080` (see `vite.config.js`). The axios `baseURL` is `/api/v1` (relative), so no CORS configuration is needed in development. For production deployment, a reverse proxy must forward `/api` to the Go backend.

**Role-aware UI:** `useAuth().isAdmin` (derived from `user.role === 'admin'`) gates admin-only buttons (Add/Edit/Delete) directly in page components. Staff users can view all pages and create invoices but cannot mutate products, customers, or invoice status.

**Tailwind component classes** are defined as `@layer components` in `src/index.css`:  
`.btn`, `.btn-primary`, `.btn-secondary`, `.btn-danger`, `.btn-sm`, `.input`, `.label`, `.card`, `.table-th`, `.table-td`  
Use these instead of repeating utility strings.

---

## Running Both Together

```bash
# Terminal 1 — backend
cd inventory-billing
go run ./cmd/server/main.go

# Terminal 2 — frontend
cd inventory-billing-ui
npm run dev
```

Open `http://localhost:5173`. First visit → go to `/register` to create an admin account, then sign in.
