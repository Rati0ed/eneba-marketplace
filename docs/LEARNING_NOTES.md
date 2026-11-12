# Learning Notes

Every new concept we use, with a short explanation and an example from our project.

---

## Architecture

### Frontend / Backend / Database
- **Frontend**: the code that runs in the user's browser (our React app).
- **Backend**: the program on a server that holds the rules and talks to the database (our Go API).
- **Database**: where data is stored permanently (our SQLite file).
- Example: the Store page (frontend) asks `GET /api/products` (backend), which runs `SELECT ... FROM products` (database).

### SPA (Single Page Application)
The browser downloads the app once. Moving between pages doesn't reload the whole site; React just swaps what's on screen.

### Monorepo
One Git repository holding several projects. Ours: `/frontend` and `/backend`.

### Deployment
Putting the app on the internet. Frontend → Vercel (serves static files). Backend → Render (runs our Go program).

---

## Web / HTTP

### REST API
A way to design a backend: each thing is a URL ("resource"), and the HTTP method says what to do with it.
- `GET` = read, `POST` = create/do an action.
- Example: `GET /api/orders` = list my orders. `POST /api/orders` = create an order.
- Python comparison: like calling functions, but over the network, e.g. `requests.get("https://.../api/orders")`.

### JSON
Text format for data. Looks almost like a Python dict:
```json
{"id": 1, "title": "Elden Ring", "price_cents": 3999}
```

### HTTP status codes
Numbers that say how a request went: `200` OK, `201` created, `400` bad input, `401` not logged in, `404` not found, `409` conflict (e.g. out of stock), `500` server bug.

### Query parameters
Extra options after `?` in a URL: `/api/products?platform=steam&sort=price_asc`. Our Store filters use these.

### CORS
Browsers block a website from calling an API on a different domain unless the API says "this site is allowed". Our Go API allows only our Vercel domain.

---

## Security

### Password hashing (bcrypt)
We never store passwords. We store a **hash**, a one-way scramble. At login we hash what the user typed and compare. bcrypt is slow on purpose, so guessing millions of passwords is expensive.

### JWT (JSON Web Token)
After login the server gives the browser a signed token like `eyJhbGc...`. It contains the user id and an expiry time, plus a signature only the server can create. The browser sends it with each request:
`Authorization: Bearer eyJhbGc...`. If someone edits the token, the signature no longer matches and the server rejects it.

### XSS (Cross-Site Scripting)
An attack where someone gets their JavaScript to run on our page, e.g. by putting `<script>` in a product name. React escapes text automatically, which protects us. This matters because our JWT sits in `localStorage`.

### Never trust the client
Anything from the browser can be faked. That's why checkout recalculates prices on the server instead of using prices the cart sent.

---

## Database

### Primary key / Foreign key
- **Primary key**: unique id of a row (`products.id`).
- **Foreign key**: a column pointing to another table's row (`offers.product_id` → `products.id`).
- SQLite needs `PRAGMA foreign_keys = ON` to enforce them.

### One-to-many relationship
One product has many offers; each offer belongs to one product.

### Transaction
A group of SQL statements that succeed **all together** or are undone **all together**. Checkout uses one so we never end up with "keys sold, order still unpaid".
Python comparison: like `with conn:` in `sqlite3`, which commits at the end or rolls back on error.

### Money as integer cents
`19.99` is stored as `1999`. Floats can't store some decimals exactly (`0.1 + 0.2` in Python gives `0.30000000000000004`).

### Seeding
Filling the database with sample data at startup so the app has something to show.

---

## Algorithms

### Levenshtein distance
The minimum number of single-letter edits (insert, delete, replace) to turn one word into another.
- `cyberpnk` → `cyberpunk`: 1 edit (insert `u`).
- `wticher` → `witcher`: 2 edits in plain Levenshtein (replace `t`→`i`, then `i`→`t`).
- **Damerau-Levenshtein** adds one rule: swapping two neighbouring letters = 1 edit. With it, `wticher` → `witcher` is 1. We use this variant.
Our search accepts small distances, so typos still find the right game.

### Debounce
Wait until the user stops typing (250 ms) before calling the API, instead of sending a request on every key press.

---

## Git

### Repository, commit, push
- **Repository (repo)**: a project folder whose full history Git tracks.
- **Commit**: a saved snapshot of the files with a message saying what changed. Like a save point in a game.
- **Push**: upload your commits to GitHub so others (and Render/Vercel) can see them.
- Example: our first commit is `chore: repo skeleton with docs, README and .gitignore`.

### .gitignore
A list of files Git must **not** track. We ignore:
- `node_modules/`: thousands of downloaded library files; anyone can recreate them with `npm install`.
- `*.db`: the SQLite file is rebuilt and seeded on every start, so it never belongs in the repo.
- `.env`: local secrets (like `JWT_SECRET`). Secrets in a public repo are leaked secrets.
- `frontend/dist/`: build output, generated from the source code.

Rule of thumb: commit what you **wrote**, ignore what a tool **generates**.

### .gitkeep
Git only tracks files, not empty folders. An empty file named `.gitkeep` keeps a folder in the repo. It's a convention, not a Git feature.

---

## Go basics (backend)

### Package and module
- A **package** is a folder of `.go` files that belong together. `db/`, `handlers/` and `middleware/` are packages. The first line of every file says which one: `package db`.
- A **module** is the whole project, defined by `backend/go.mod`. It has a name (`github.com/Rati0ed/eneba-marketplace/backend`) and lists the libraries we use with exact versions.
- `go.sum` stores a checksum of every downloaded library, so nobody can swap a library for a tampered copy.
- Python comparison: package ≈ Python package (a folder with `__init__.py`), `go.mod` ≈ `requirements.txt` with pinned versions.

### Capital letter = public
In Go, a name starting with a **capital letter** can be used from other packages; lowercase is private to its package.
- `db.Open(...)` works from `main.go` because `Open` is capitalized.
- `slugify` and `seed` are lowercase, so only code inside `db/` can call them.
Python has no real equivalent (just the `_private` naming convention); in Go the compiler enforces it.

### Structs and methods
A **struct** groups fields, like a Python class with only attributes (or a dataclass):
```go
type Handler struct {
    DB *sql.DB
}
```
A **method** is a function attached to a type. `(h *Handler)` is the receiver, like `self` in Python:
```go
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) { ... h.DB ... }
```

### Errors are values
Go has no exceptions. Functions return an error as an extra value, and you check it right away:
```go
database, err := db.Open(dbPath)
if err != nil {
    log.Fatalf("database: %v", err)
}
```
Python comparison: instead of `try/except` around a call, every call that can fail returns `(result, err)`.
`fmt.Errorf("seed data: %w", err)` wraps an error with context, so the log shows *where* it failed: `seed data: insert product "x": UNIQUE constraint failed`.

### `:=` vs `=`
- `x := 5` declares a new variable and sets it (type is inferred).
- `x = 6` changes an existing variable.

### `defer`
Runs a line when the surrounding function **ends**, however it ends (normal return or early error return).
```go
rows, err := h.DB.Query(...)
defer rows.Close()   // guaranteed cleanup, like Python's `with` block
```

### Pointers (`*int`, `&x`)
A pointer is the address of a value. We use `*int` for `LowestPriceCents` because a pointer can be `nil`, which means "no value" (JSON `null`). A plain `int` is always a number; it would show `0`, which would wrongly look like "free".
`&p.ID` in `rows.Scan(&p.ID, ...)` passes the *address*, so Scan can write into our variable.

### Slices and maps
- Slice `[]ProductCard` ≈ Python list. `append(items, p)` ≈ `items.append(p)`.
- Map `map[string]bool` ≈ Python dict. We use one as a set in the CORS middleware.
- `make([]ProductCard, 0)` creates an empty (not nil) slice, so JSON gets `[]` instead of `null`.

### `//go:embed`
Copies a file into the compiled program at build time:
```go
//go:embed schema.sql
var schemaSQL string
```
The server is a single file that doesn't need `schema.sql` next to it at runtime.

### Struct tags
`json:"lowest_price_cents"` after a field tells the JSON encoder what key name to use. Without it the key would be the Go name `LowestPriceCents`.

---

## HTTP server in Go (`net/http`)

### Handler
A function that receives a request and writes a response:
```go
func (h *Handler) Health(w http.ResponseWriter, r *http.Request)
```
- `r` = the incoming request (method, URL, headers, body).
- `w` = where we write the answer (status code, headers, body).
Python/Flask comparison: like a function decorated with `@app.get("/api/health")`, except Flask returns the response while Go writes it into `w`.

### Router (ServeMux)
Maps "method + path" to handlers (`main.go`):
```go
mux.HandleFunc("GET /api/products", h.ListProducts)
mux.HandleFunc("GET /api/covers/{file}", h.Cover)   // {file} is a wildcard: r.PathValue("file")
```
Go's router handles wrong methods automatically: `POST /api/products` gets `405 Method Not Allowed`.

### Middleware
A wrapper around a handler that runs code before and after it, for **every** request. Like a Python decorator. Ours:
- `Logging`: prints `GET /api/products 200 3.2ms`.
- `CORS`: adds the headers that let our frontend call the API.
```go
handler := middleware.Logging(middleware.CORS(allowedOrigin, mux))
```
The request goes Logging → CORS → router → handler, and the response comes back the same way.

### CORS preflight
Before a "non-simple" request (e.g. `POST` with JSON, or one with an `Authorization` header), the browser first sends an `OPTIONS` request asking "may I?". Our CORS middleware answers `204 No Content` with the allow-headers. If the origin isn't on our list, we add no headers and the **browser** blocks the response.
Note: CORS is enforced by browsers. `curl` ignores it, so CORS is not a replacement for authentication.

### Server timeouts
`ReadTimeout`, `WriteTimeout` and the rest stop a slow or malicious client from holding a connection open forever (a "Slowloris" attack). Go's default is *no* timeout, so we set them.

### Environment variables
Settings given to a program from outside, not written in the code. Same code, different settings per place:
| Variable | Local default | On Render |
|---|---|---|
| `PORT` | 8080 | set automatically by Render |
| `ALLOWED_ORIGIN` | http://localhost:5173 | our Vercel URL |
| `JWT_SECRET` | (empty, warning) | long random string |
| `DB_PATH` | data.db | default is fine |
Secrets must live in env vars, never in Git. Python: `os.environ.get("PORT", "8080")`; ours is `getEnv("PORT", "8080")`.

---

## SQL used in the backend

### LEFT JOIN + GROUP BY + MIN/COUNT
From `handlers/products.go`. One row per product, with its cheapest in-stock offer:
```sql
SELECT p.id, ..., MIN(o.price_cents), COUNT(o.id)
FROM products p
LEFT JOIN offers o ON o.product_id = p.id AND EXISTS (...available key...)
GROUP BY p.id
```
- `JOIN` pairs each product with each matching offer (a product with 4 offers → 4 rows).
- `LEFT` keeps products with **no** matching offer (their offer columns are NULL).
- `GROUP BY p.id` folds those rows back to one per product; `MIN`/`COUNT` summarize the group.
- `COUNT(o.id)` counts only non-NULL values, so a product with no offers gets 0, not 1.

### EXISTS
`EXISTS (SELECT 1 FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available')` is true if at least one such row exists. It stops at the first match, so it's cheap. We use it to ignore sold-out offers.

### Placeholders `?` (prevents SQL injection)
```go
tx.Exec(`INSERT INTO sellers (name, ...) VALUES (?, ...)`, s.Name, ...)
```
The values are sent **separately** from the SQL text, so a value like `'; DROP TABLE users; --` is stored as plain text, never run as SQL. **Never** build SQL by gluing strings together with user input.
Python's `sqlite3` works the same: `cur.execute("... WHERE id = ?", (id,))`.

### CHECK and UNIQUE constraints
Rules the database enforces itself, even if our Go code has a bug:
- `CHECK (price_cents > 0)`: no free or negative prices.
- `CHECK (status IN ('available', 'sold'))`: no typos like `'sodl'`.
- `UNIQUE (product_id, seller_id)`: a seller can't list the same product twice.

### Index
`CREATE INDEX idx_offers_product ON offers(product_id)` is like a book's index: finding "offers of product 7" jumps straight to them instead of reading the whole table.

### WAL mode and busy_timeout
- **WAL** (write-ahead log): readers and the writer don't block each other, so the API can serve the catalog while a checkout writes.
- **busy_timeout=5000**: if the database is locked, wait up to 5 s instead of failing instantly.

### PRAGMA foreign_keys is per connection
SQLite turns foreign keys on **per connection**, and Go's `database/sql` keeps a **pool** of several connections. Running `PRAGMA foreign_keys = ON` once would only affect one of them. That's why we put `_foreign_keys=on` in the connection string (`db/db.go`): the driver applies it to every new connection. `TestForeignKeysAreEnforced` proves it.

### LastInsertId
After an `INSERT`, `res.LastInsertId()` returns the id SQLite gave the new row. We use it to link offers to the product we just inserted.

---

## Testing in Go

- Test files end in `_test.go`; test functions are `func TestXxx(t *testing.T)`. Run all with `go test ./...`.
- `t.Errorf` = report a failure and continue; `t.Fatalf` = report and stop this test.
- **Table-driven tests**: a list of input → expected output, checked in one loop (`TestSlugify`, `TestCoverName`). Adding a case is one line.
- `t.TempDir()` gives a throwaway folder, so each test gets its own fresh database.
- Python comparison: like `pytest` with `@pytest.mark.parametrize`.

---

## Security details in the backend

### Escaping output (SVG covers)
`handlers/covers.go` puts product titles into SVG markup. `html.EscapeString` turns `<`, `&`, `'` into `&lt;`, `&amp;`, `&#39;`, so a title like `<script>` is shown as text and can't become markup. It's the same idea as SQL placeholders: **data must never be read as code**.

### Don't leak internal errors
`serverError` logs the real error for us but sends the client only `"something went wrong"`. Details like SQL text or file paths help attackers.

---

## Frontend tooling

### npm and package.json
- **npm** downloads JavaScript libraries (like `pip`).
- `package.json` lists our libraries and scripts (like `requirements.txt` + a Makefile):
  - `dependencies`: needed in the browser: `react`, `react-dom`.
  - `devDependencies`: only needed to build/check: `vite`, `typescript`, `oxlint`.
  - `scripts`: `npm run dev` (dev server), `npm run build` (check types + build), `npm run lint` (find bugs).
- `package-lock.json` pins the **exact** versions of every library, so Vercel installs the same versions we tested.
- `^19.3.1` means "19.3.1 or a newer 19.x"; the lock file decides the exact one.

### Vite
The tool that runs and builds our frontend.
- `npm run dev`: a local dev server on http://localhost:5173. Saving a file updates the page instantly.
- `npm run build`: turns our TypeScript/React files into plain HTML, CSS and JS in `dist/`. Browsers can't run TypeScript, so this step is needed. Vercel runs this and serves `dist/`.

### VITE_ environment variables are baked in at build time
`import.meta.env.VITE_API_URL` is **replaced by its value when the app is built** (in `src/api.ts`). Two consequences:
1. After changing `VITE_API_URL` on Vercel you must **redeploy**; the old build still has the old value.
2. Anything in a `VITE_` variable is visible to everyone who opens the site. **Never put secrets there.** (The API URL is fine; it's public anyway.)
Only variables starting with `VITE_` are exposed, so other variables on the build machine don't leak into the app by accident.

---

## TypeScript

TypeScript = JavaScript + types. The types are checked when building (`tsc`), then removed; the browser runs plain JavaScript.

### Types for our data
```ts
export type ProductCard = {
  id: number
  title: string
  lowest_price_cents: number | null   // "number or null"
}
```
If we write `product.titel`, the build fails instead of showing `undefined` on the page. Python comparison: type hints, but actually enforced.

### `number | null` (union type)
A value that can be one of several types. TypeScript forces us to handle `null` before using it as a number. That's why `App.tsx` checks `product.lowest_price_cents === null` before `formatPrice(...)`.

### Generics `<T>`
```ts
async function apiGet<T>(path: string): Promise<T>
apiGet<{ items: ProductCard[] }>('/api/products')
```
`T` is a type parameter: the caller fills it in. One function works for every endpoint and still knows the return type.

### `async` / `await` / Promise
Network requests take time. A **Promise** is "a value that will arrive later". `await` waits for it without freezing the page.
```ts
const response = await fetch(API_URL + path)
```
Python comparison: the same `async`/`await` as in `asyncio`.

### `fetch`
The browser's built-in function for HTTP requests (like Python `requests.get`). It does **not** throw on 404/500, only on network failure, so we check `response.ok` ourselves in `apiGet`.

---

## React

### Component and JSX
A component is a function that returns what to show. JSX is the HTML-like syntax inside it:
```tsx
export default function App() {
  return <h1>Store</h1>
}
```
Differences from HTML: `className` instead of `class`; `{...}` inserts a TypeScript value: `{product.title}`.
React escapes everything inserted with `{}`, so a product title containing `<script>` is shown as text. That's our main XSS protection.

### useState: memory of a component
```tsx
const [loading, setLoading] = useState(true)
```
`loading` is the current value; `setLoading(false)` changes it **and re-renders** the component. A normal variable would reset every time the function runs, and changing it wouldn't update the screen.

### useEffect: do something after showing
```tsx
useEffect(() => {
  fetchProducts(...).then(setProducts)
  return () => controller.abort()   // cleanup
}, [])                               // [] = run once, when the component first appears
```
Rendering must only *describe* the page. Side effects like network requests go in `useEffect`. The returned function is the **cleanup**, which runs when the component disappears.

### Loading / error / success states
Every request can be slow or fail. `App.tsx` keeps three pieces of state (`loading`, `error`, `products`) and shows a different UI for each. Without this, users see a blank page while waiting.

### Lists and `key`
```tsx
{products.map((p) => <li key={p.id}>...</li>)}
```
`.map` turns data into elements (like a Python list comprehension). `key` must be unique and stable; React uses it to know which item is which when the list changes.

### Conditional rendering
- `{loading && <p>Loading…</p>}`: shows the `<p>` only if `loading` is true.
- `{cond ? <A/> : <B/>}`: if/else inside JSX.

### StrictMode
`main.tsx` wraps the app in `<StrictMode>`. In development only, React runs effects twice to expose missing cleanups. That's why our request uses an `AbortController`: the first request is cancelled cleanly.

### AbortController
Cancels a `fetch` that's in progress. We cancel in the effect's cleanup, so a late answer never updates a component that's gone.
