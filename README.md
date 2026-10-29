# StockFlow — Inventory & Sales

A small inventory and sales module: manage products, adjust stock, and record sales orders that consume stock.

- **Backend:** Go (`net/http`, `database/sql` + pgx) and PostgreSQL
- **Frontend:** React + TypeScript (Vite), calling the real API
- **Infra:** Docker Compose runs the whole stack (db → migrations → seed → API → web)

---

## Quick start (one command)

Requirements: Docker with Compose v2.

```bash
git clone https://github.com/ahmadfrnando/stockoverflow.git
cd stockoverflow
docker compose up -d --build
```

Then open:

| What | URL |
|---|---|
| Web app | http://localhost:3000 |
| API | http://localhost:8080/api/products |
| Health | http://localhost:8080/health |

What happens on `docker compose up`:

1. `db` starts Postgres 16 and waits until it is healthy.
2. `migrate` applies the SQL migrations in `backend/db/migrations`, then exits.
3. `seed` inserts 13 products (with opening-balance stock movements) and 3 sample orders (one cancelled), then exits. It is idempotent, so running `up` again does not duplicate data.
4. `api` starts the Go server.
5. `web` serves the built frontend with nginx and proxies `/api` to the `api` container.

`migrate` and `seed` showing `Exited (0)` in `docker compose ps` is expected.

### Configuration

Everything works with the defaults. To override, copy `.env.example` to `.env` and set what you need:

| Variable | Default | Purpose |
|---|---|---|
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | `postgres` / `postgres` / `stackoverflow` | Database credentials |
| `DB_PORT` | `5432` | Postgres port on the host |
| `APP_PORT` | `8080` | API port on the host |
| `WEB_PORT` | `3000` | Web app port on the host (change it if 3000 is taken) |
| `TEST_DATABASE_URL` | — | Only used by backend tests (see below) |

To start from an empty database: `docker compose down -v && docker compose up -d --build`.

---

## Running tests

### Backend (integration tests against real Postgres)

The tests run against a **separate** database so they never touch the app data. They read only `TEST_DATABASE_URL` and are **skipped** when it is not set.

Requirements: Go 1.27+.

```bash
# 1. create the test database and migrate it (uses the running db container)
docker compose exec db psql -U postgres -c "CREATE DATABASE stockoverflow_test;"
docker compose run --rm migrate \
  -path=/migrations \
  -database="postgres://postgres:postgres@db:5432/stockoverflow_test?sslmode=disable" up

# 2. run the tests
cd backend
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/stockoverflow_test?sslmode=disable" \
  go test -p 1 -race ./...
```

`-p 1` runs packages one at a time, because each test truncates the shared test database.

What is covered (`backend/internal/order/repository_test.go`):

| Test | Proves |
|---|---|
| `TestCreate_InsufficientStock_WritesNothing` | An order where one item is short is rejected with `InsufficientStockError` (correct SKU and available qty), and **nothing** is written: the other item's stock is rolled back, and there are no orders, items, or movements. |
| `TestCreate_HappyPath_UpdatesStockAndWritesMovements` | A valid order decrements stock, returns the server-computed total, and writes one `out` movement per item. |
| `TestCreate_Concurrent_NoOversell` | 10 concurrent orders for a SKU with stock 5: exactly 5 succeed, 5 get insufficient stock, the final stock is 0, and 5 orders exist. Runs under `-race`. |

### Frontend (Vitest, pure logic)

Requirements: Node 22+.

```bash
cd frontend
npm ci
npm test
```

Covers rupiah formatting, "2h ago" formatting, qty validation (integer > 0), and order form validation, including mapping a 409 to the exact row that failed.

---

## Local development (without the web/api containers)

```bash
# database only
docker compose up -d db migrate seed

# API on :8080
cd backend
DB_USER=postgres DB_PASSWORD=postgres DB_NAME=stackoverflow go run ./cmd/api

# frontend on :5173 (Vite proxies /api to the API)
cd frontend
npm ci
npm run dev                                   # proxies to http://localhost:8080
API_URL=http://localhost:8010 npm run dev     # or point the proxy at another port
```

**Proxy vs CORS:** I chose the Vite dev proxy (`frontend/vite.config.ts`) over CORS middleware. The browser only ever talks to one origin, so the Go API needs no CORS code. In Docker, nginx does the same job (`frontend/nginx.conf`).

---

## API

All errors use one envelope:

```json
{ "error": { "code": "INSUFFICIENT_STOCK", "message": "...", "fields": {}, "sku": "SKU-001", "available": 3 } }
```

`code` is always present. The other keys appear only when relevant.

| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/api/products?search=&status=&page=&limit=` | 200 `{data, page, limit, total}` | 400 invalid `status` |
| GET | `/api/products/summary` | 200 `{data: {total_products, active_products, total_stock_value, low_stock_count}}` | |
| GET | `/api/products/{id}` | 200 `{data}` | 404 |
| POST | `/api/products` | 201 `{data}` | 400 `VALIDATION_FAILED`, 409 `DUPLICATE_SKU` |
| PUT | `/api/products/{id}` | 200 `{data}` | 400, 404 |
| DELETE | `/api/products/{id}` | 204 | 404, 409 `PRODUCT_HAS_HISTORY` |
| POST | `/api/products/{id}/stock` | 201 `{data: {movement, stock}}` | 400, 404, 409 `INSUFFICIENT_STOCK` + `available` |
| GET | `/api/products/{id}/movements` | 200 `{data: [...]}`, newest first | 404 |
| POST | `/api/orders` | 201 `{data}` with items, unit prices, and total | 400 `VALIDATION_FAILED` + `fields`, 409 `INSUFFICIENT_STOCK` + `sku` + `available` |
| GET | `/api/orders/{id}` | 200 `{data}` | 404 |
| POST | `/api/orders/{id}/cancel` | 200 `{data}` | 404, 409 `ALREADY_CANCELLED` |

`status` filter values: `active`, `inactive`, `low_stock` (stock ≤ 10).

### curl examples

```bash
# list and search
curl "localhost:8080/api/products?search=kopi&page=1&limit=10"
curl "localhost:8080/api/products?status=low_stock"
curl "localhost:8080/api/products/summary"

# create / update / delete a product
curl -X POST localhost:8080/api/products -H 'Content-Type: application/json' \
  -d '{"sku":"SKU-100","name":"Teh Melati 25 Pcs","price":15000}'
curl -X PUT localhost:8080/api/products/14 -H 'Content-Type: application/json' \
  -d '{"name":"Teh Melati 30 Pcs","price":17000,"is_active":true}'
curl -i -X DELETE localhost:8080/api/products/14       # 204
curl -i -X DELETE localhost:8080/api/products/1        # 409 PRODUCT_HAS_HISTORY

# adjust stock
curl -X POST localhost:8080/api/products/12/stock -H 'Content-Type: application/json' \
  -d '{"type":"in","qty":10,"note":"restock"}'
curl -X POST localhost:8080/api/products/10/stock -H 'Content-Type: application/json' \
  -d '{"type":"out","qty":1}'                          # 409 INSUFFICIENT_STOCK, available 0
curl localhost:8080/api/products/12/movements

# orders
curl -X POST localhost:8080/api/orders -H 'Content-Type: application/json' \
  -d '{"customer":"Budi","items":[{"sku":"SKU-001","qty":2},{"sku":"SKU-006","qty":1}]}'
curl -X POST localhost:8080/api/orders -H 'Content-Type: application/json' \
  -d '{"customer":"Budi","items":[{"sku":"SKU-010","qty":1}]}'   # 409, sku SKU-010, available 0
curl -X POST localhost:8080/api/orders -H 'Content-Type: application/json' \
  -d '{"customer":"","items":[{"sku":"SKU-001","qty":0},{"sku":"sku-001","qty":1}]}'   # 400 with fields
curl localhost:8080/api/orders/1
curl -X POST localhost:8080/api/orders/1/cancel        # 200, stock restored
curl -X POST localhost:8080/api/orders/1/cancel        # 409 ALREADY_CANCELLED

# concurrency check: 10 parallel orders against SKU-007 (stock 25) at qty 5 → 5 succeed, 5 get 409
for i in $(seq 1 10); do
  curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost:8080/api/orders \
    -H 'Content-Type: application/json' \
    -d '{"customer":"race","items":[{"sku":"SKU-007","qty":5}]}' &
done; wait
```

---

## Design decisions

### Data safety under concurrency

- **Guarded UPDATE, never SELECT-then-UPDATE.** Stock is decremented with
  `UPDATE products SET stock = stock - $1 WHERE sku = $2 AND stock >= $1 RETURNING price`.
  The check and the write are one atomic statement. Concurrent updates on the same row are serialized by Postgres's row lock, and the `WHERE` is re-checked against the committed value, so stock can never go negative. `0 rows` means insufficient stock. Only then does a follow-up `SELECT` read the available qty for the error message.
- **One transaction per order.** Validation, every stock decrement, the order row, the item rows, and the `out` movements all happen in one transaction. Any failure rolls everything back, which `TestCreate_InsufficientStock_WritesNothing` proves.
- **Deadlock avoidance.** Order items are sorted by SKU before the updates (and cancel reads items `ORDER BY sku`), so concurrent transactions always lock rows in the same order.
- **Cancel is guarded too.** `UPDATE sales_orders SET status = 'CANCELLED' ... WHERE id = $1 AND status != 'CANCELLED'` makes a double cancel impossible, even if two requests race.
- **Stock adjustments** use the same pattern (`WHERE id = $2 AND stock + $1 >= 0`). The movement row is written in the same transaction.

### Data model

- **`last_movement_at` is derived, not stored.** It is computed with a correlated subquery, `MAX(stock_movements.created_at)`, backed by a new index on `stock_movements (product_id, created_at DESC)`. This keeps a single source of truth, so no code path can forget to update it. If the table grows very large, or if the list needs to sort or filter by it, I would switch to a column maintained in the same transaction as each movement.
- **Stock only changes through movements.** New products start at 0, and `PUT` cannot edit stock. Seeded stock is recorded as an "Opening balance" `in` movement. Orders write `out` movements, and cancellations write `in` movements. This keeps stock fully auditable.
- **Hard delete guarded by the database.** `DELETE` is a real delete. When a product has movement history, the foreign key violation (`23503`) becomes `409 PRODUCT_HAS_HISTORY`. The database enforces it, so there is no check-then-delete race. Deactivate a product with `PUT {"is_active": false}`.
- **SKU is immutable** after creation, because order items reference it.
- **Money is an integer** (IDR). There is no float math anywhere.

### API behaviour

- **The client never sends prices.** `unit_price` is read from `products` inside the decrementing `UPDATE ... RETURNING price`, and `total` is computed on the server.
- **400 before any write.** Empty items, a missing customer, qty ≤ 0, and duplicate SKUs (case-insensitive) are caught in the service. Unknown or inactive SKUs are caught in the transaction before any `UPDATE`. All are reported per field (`items[1].qty`, ...).
- **Layering:** `handler` (HTTP and JSON) → `service` (validation and normalization) → `repository` (SQL and transactions), wired by hand in `cmd/api/main.go`. Errors flow up as sentinel errors or typed errors (`ValidationError`, `InsufficientStockError`) and are mapped to status codes only in the handler.

### Frontend

- `src/api/types.ts` holds the shared API types and mirrors the Go JSON. There is no `any` on domain data.
- `src/api/client.ts` is the only module that calls `fetch`. It builds every URL, unwraps `{data}`, and turns every non-2xx response into an `ApiError` that carries the parsed error envelope.
- Cross-stack errors are shown in the UI, never only in the console:
  - The adjust-stock 409 shows "only N available".
  - An order 409 highlights the exact row whose SKU failed.
  - Order 400 `fields` use the same keys as client-side validation, so one renderer displays both.
- Every mutation re-fetches from the server. This includes the product list and summary cards after adjust, and the list after create or cancel when navigating back. The UI never patches numbers locally.
- Products page states: **loading** (on first load and retry), **empty** (search for `zzz`), and **error** with Retry (`docker compose stop api`, then click Retry after `docker compose start api`).
- Search is debounced through a small `useDebounce` hook. The status filter and the summary are computed on the server, so they are correct across pages.

---

## What I would improve next

- **Idempotency-Key on `POST /api/orders`**, so client retries can never create a double order.
- **Routing with URLs** (react-router) so order pages can be refreshed and shared. Currently navigation is in-memory state.
- **An order list page** and a way to browse orders. Currently orders are reached after creating one.
- **Product create/edit/delete UI.** The API supports it, but only adjust stock is exposed in the UI.
- **Pagination controls** in the products table. The API already paginates, and the UI shows the first 20.
- **More backend tests:** cancel (stock restored and the double cancel race), stock adjustment, and HTTP-level handler tests for the error envelope.
- **Hardening:** server read/write timeouts, `http.MaxBytesReader` on request bodies, structured logging, CI running both test suites, and a health check on the `api` container.
- **Daily sales report** endpoint and summary card (a bonus item not done).

---

## Hours spent

<!-- TODO: fill in honestly before submitting -->
About **X hours** of focused work, spread over 28–29 October 2026.

## How this was built

<!-- TODO: adjust this to your own words -->
This was my first project in Go. I wrote the code myself and used an AI as a mentor and references.
