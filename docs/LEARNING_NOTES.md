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

See **Day 3: Search** below for edit distance, debounce and the full search algorithm.

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

---

## Day 2: Store (backend)

### Pagination with LIMIT / OFFSET
```sql
... ORDER BY sales_count DESC LIMIT 24 OFFSET 24   -- page 2
```
`LIMIT` = how many rows, `OFFSET` = how many to skip. Page `n` skips `(n-1) * 24`. A second query `SELECT COUNT(*)` with the **same WHERE** gives `total`, and `total_pages = ceil(total / 24)`.
The page size is fixed on the server (`pageSize` in `handlers/products.go`), so nobody can request a million rows.

### CTE (`WITH ... AS`)
A named sub-query you can use like a table (`handlers/cards.go`):
```sql
WITH cards AS (SELECT p.*, MIN(o.price_cents) AS lowest_price_cents ... GROUP BY p.id)
SELECT * FROM cards WHERE lowest_price_cents <= ? ORDER BY ...
```
We need it because you can't filter on `MIN(...)` with a normal `WHERE` (WHERE runs before grouping). Inside the CTE the minimum is calculated; outside it's just a column.

### Building a WHERE clause safely
`buildWhere` adds a condition only for filters the user chose, and keeps **SQL text** and **values** separate:
```go
"WHERE platform IN (?, ?) AND lowest_price_cents >= ?"   // text: written by us
[]any{"Steam", "Xbox", 1000}                              // values: from the user
```
The number of `?` is generated (`placeholders(2)` → `"?, ?"`), but the user's text never enters the SQL string.

### Allow-list (when placeholders can't be used)
`ORDER BY` and column names can't be `?` placeholders. So the user sends a **key** (`sort=price_asc`) and we look up SQL we wrote ourselves in `sortOptions`. Anything not in the map is rejected with 400. `?sort=price; DROP TABLE products` is just an unknown key. A test checks this (`TestListProductsRejectsBadInput`).

### Validate input, answer 400
Bad input (`page=abc`, `min_price=-5`, `min_price > max_price`) gets `400 Bad Request` with a message saying what's wrong, instead of a 500 or silently wrong results. Rule: **400 = the client's mistake, 500 = our bug.**

### Repeated query parameters
`?platform=Steam&platform=Xbox` → in Go, `query["platform"]` gives `["Steam", "Xbox"]` (`query.Get` would only give the first). In the browser, `URLSearchParams.getAll('platform')` does the same.

### COALESCE
`COALESCE(a, b)` returns `a` unless it's NULL, then `b`. Used in `ProductFilters` so an empty table gives a price range of 0 instead of NULL.

### Testing handlers with httptest
`httptest.NewRequest` builds a fake request and `httptest.NewRecorder` is a fake `ResponseWriter` that remembers the status and body. We call the handler function directly and check the JSON: no real server, no network, fast (`handlers/products_test.go`).

---

## Day 2: Store (frontend)

### React Router
Lets one React app have many pages with real URLs (`App.tsx`):
```tsx
<Route path="/store" element={<StorePage />} />
<Route path="/product/:slug" element={<ProductPage />} />
```
- `<Link to="/store">` changes the URL **without reloading** the page.
- `useParams()` reads `:slug` from the URL.
- `<Navigate to="/store" />` redirects.

### Why vercel.json has a "rewrite"
On a refresh at `/store`, the browser asks Vercel for a file called `/store`, which doesn't exist (there is only `index.html`). The rewrite says "for any path, send `index.html`"; then React Router reads the URL and shows the right page. Without it, refreshing any page except `/` gives a 404.

### The URL as state (`useSearchParams`)
On the store, **the URL is the only place filters are stored**:
```
/store?platform=Steam&region=Europe&sort=price_asc&page=2
```
Clicking a filter changes the URL, and the URL change triggers the fetch (`useEffect` depends on `queryString`). Benefits: back/forward buttons work, refresh keeps filters, links can be shared. And the store URL params equal the API params, so we pass `searchParams.toString()` straight to `/api/products`.

### Derived state (don't store what you can calculate)
`StorePage` doesn't keep a `loading` flag. It remembers which request its results belong to (`loaded.key`) and calculates:
```ts
const loading = loaded.key !== requestKey
```
Two separate states (`loading` + `result`) can get out of sync; a calculation can't. Same idea: `selected` filters and `chips` are calculated from the URL on every render, not stored.

### Race conditions and AbortController
Click "Steam", then quickly "Xbox": two requests run. If the Steam answer arrives **last**, it would overwrite the Xbox results. The effect's cleanup aborts the old request when the URL changes, so only the newest answer is used.

### Props and "lifting state up"
Props are a component's inputs, like function arguments: `<Pagination page={2} totalPages={4} onChange={setPage} />`.
`FilterSidebar` doesn't change the URL itself. `StorePage` owns the state and passes down **values** (`selected`) and **functions** (`onToggle`). One owner = one source of truth.

### The `key` trick to reset a component
`<PriceFilter key={`${minPrice}-${maxPrice}`} ... />`: when `key` changes, React throws the old component away and creates a fresh one. We use it so the price inputs reset after "Clear filters", while still letting the user type freely (local state) without a request per keystroke.

### Loading skeletons
Grey placeholder cards with the same size as real ones (`ProductCardSkeleton`). The page has its final layout immediately, so nothing jumps when data arrives. On later filter changes we keep the old cards visible but faded (`.refreshing`), which flickers less.

---

## CSS used for the design

- **CSS variables** (`index.css`): `--purple: #4618ac` defined once, used as `var(--purple)`. The whole theme is in one place.
- **Flexbox** (`display: flex`): lines things up in a row or column, e.g. the header: logo | search | nav.
- **Grid** (`display: grid`): 2D layouts. The product grid uses `repeat(auto-fill, minmax(180px, 1fr))`: "as many columns of at least 180px as fit". It's responsive without media queries.
- **Media queries** (`@media (max-width: 900px)`): different rules for small screens. The sidebar becomes a toggle panel.
- **`position: sticky`**: the header and sidebar stay visible while scrolling.
- **`aspect-ratio: 3 / 4`**: reserves the image box before the image loads (no layout jump).
- **`-webkit-line-clamp: 2`**: titles cut to 2 lines with "…".
- **`:focus-visible`**: a yellow outline for keyboard users only. Accessibility.
- **`prefers-reduced-motion`**: turns off animations for users who disabled them in their OS.

---

## Day 3: Search (backend)

### Edit distance (Damerau-Levenshtein)
The fewest single-letter edits to turn one word into another. Our edits: **insert, delete, replace, swap two neighbours** (`search/search.go`, `editDistance`).
- `eldn` → `elden`: 1 (insert e)
- `wticher` → `witcher`: 1 (swap t,i). Plain Levenshtein says 2; swaps are the most common typing mistake, so we count them as 1.
- `kitten` → `sitting`: 3

**How it's computed: dynamic programming.** Fill a table where `dist[i][j]` = distance between the first `i` letters of word A and the first `j` letters of word B. Each cell is the cheapest of:
```
dist[i-1][j] + 1          delete a letter
dist[i][j-1] + 1          insert a letter
dist[i-1][j-1] + 0 or 1   keep (same letter) or replace
dist[i-2][j-2] + 1        swap (if the last two letters are swapped)
```
The bottom-right cell is the answer. Cost: `len(A) × len(B)` steps, tiny for words.

### Runes vs bytes
In Go a `string` is bytes. `"é"` is 2 bytes but 1 letter. `[]rune("pokémon")` splits into real letters, so the distance between `pokémon` and `pokemon` is 1, not 2. Python 3 strings are already letters, so Python doesn't have this problem.

### Tokenizing and normalizing
Turning text into comparable words: lowercase, remove accents, split on anything that's not a letter or digit (`strings.FieldsFunc` + `unicode.IsLetter`). Both the titles and the query go through the **same** `normalize`, so they're comparable.

### Scoring and ranking
Each query word gets points for its best match (exact 10, prefix 7, 1 typo 5, ...). Every word must match (**AND**), otherwise the product is out. Sort by total score, ties by popularity (`sort.Slice` with a custom "less" function).
Results under half the best score are dropped: precision matters as much as tolerance. A search that returns everything is as useless as one that returns nothing.

### In-memory index
`search.NewIndex` pre-processes every title once at startup. Each search loops over 80 products in microseconds. A real shop with millions of products would use a search engine (Elasticsearch, Meilisearch) with an inverted index; for 80 products that would be extra infrastructure for no benefit.

### Pure package = easy tests
The `search` package doesn't know about HTTP or SQL: titles in, IDs out. So its tests need no database. The test table in `search_test.go` is the **spec**: each line is "what people type" → "what must come first".

### Ordering SQL rows by our own ranking
The search decides the order, but SQL fetches the rows. `relevanceOrder` builds:
```sql
ORDER BY CASE id WHEN ? THEN 0 WHEN ? THEN 1 WHEN ? THEN 2 END
```
with the product ids as values, so SQL returns them in search order while still applying filters and pages.

---

## Day 3: Search (frontend)

### Debounce
Wait until the user stops typing for 250 ms, then search once (`SearchBar.tsx`):
```ts
useEffect(() => {
  const timer = setTimeout(() => fetchSuggestions(query, ...), 250)
  return () => { clearTimeout(timer); controller.abort() }   // cleanup
}, [query])
```
Every keystroke changes `query`, so React runs the cleanup (cancel the previous timer) and starts a new timer. Only the last one survives. Typing `elden` = **1 request** instead of 5 (the browser test checks this).

### useRef
`useRef` gives direct access to a DOM element: `inputRef.current?.blur()` removes focus from the input (closing the phone keyboard). Unlike state, changing a ref doesn't re-render.

### Keyboard navigation
`activeIndex` remembers the highlighted result (-1 = none). ↓ adds 1, ↑ subtracts 1, and `%` (modulo) wraps around: `(index + 1) % length`. Enter opens the highlighted product, or searches the store if none is highlighted. `event.preventDefault()` stops the browser's default action (moving the cursor, submitting the form, clearing the search box on Esc).

### Blur vs click (onMouseDown + preventDefault)
Clicking a result would first blur the input, closing the dropdown **before** the click lands. `onMouseDown={(e) => e.preventDefault()}` on the dropdown keeps focus in the input, so the click works.

### ARIA combobox
Attributes that tell screen readers what's going on: `role="combobox"` on the input, `role="listbox"` and `role="option"` on the list, `aria-expanded` (is the list open?), `aria-activedescendant` (which option is highlighted). The browser test checks `aria-activedescendant` follows the arrow keys.

### Escaping user input in a regular expression
`Highlight.tsx` builds a RegExp from the user's words. Characters like `+ ( [ .` mean something in a regex, so `c++ (pc` would crash the page. We escape them first: `word.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')`. Same principle as SQL placeholders and HTML escaping: **user input must never be treated as code**, in any language.

### URL-encoding
`new URLSearchParams({ q: 'gta 5 & more' })` → `q=gta+5+%26+more`. Raw `&` would start a new parameter. Never glue user text into a URL by hand.

### Resetting a component with `key` (again)
`<SearchBar key={urlQuery} initialQuery={urlQuery} />` in `Header.tsx`: when the URL's `?q=` changes (a new search, or the back button), the search bar is re-created with the new text and a closed dropdown. Same trick as the price filter on Day 2.

---

## Day 4: Product page and homepage (backend)

### Path parameters and route precedence
`mux.HandleFunc("GET /api/products/{slug}", h.GetProduct)` → inside the handler `r.PathValue("slug")`. When two patterns match the same URL, Go picks the **more specific** one: `/api/products/filters` beats `/api/products/{slug}`. A test checks this (`TestProductPageEndpoints`).

### sql.ErrNoRows → 404
`QueryRow(...).Scan(...)` returns `sql.ErrNoRows` when nothing matched. That's the client asking for something that doesn't exist (**404**), not a server failure (500). We check it with `errors.Is(err, sql.ErrNoRows)`.

### Correlated sub-query
```sql
SELECT o.id, ..., (SELECT COUNT(*) FROM game_keys k WHERE k.offer_id = o.id AND k.status = 'available') AS stock
FROM offers o ...
```
The inner SELECT refers to `o` from the outer query, so it runs once per offer and counts that offer's keys (`handlers/product.go`).

### Struct embedding
`type ProductDetail struct { ProductCard; Description string }`: `ProductDetail` gets every `ProductCard` field, and the JSON stays flat. Go's alternative to inheritance: composition.

### Data-driven homepage
The homepage layout is **data**, not code. `homepage_blocks` rows say which block types to show, in which order, and which endpoint feeds each one. `TestHomeBlocksComeFromTheDatabase` changes the table (hides a block, moves and renames another) and checks that `/api/home/blocks` follows. No frontend change needed.

### Routes in one place
`handlers/routes.go` builds the router; `main.go` and the tests both use `h.Routes()`. Tests therefore go through the same routing as production (`call()` helper in `home_test.go`).

---

## Day 4: Product page and homepage (frontend)

### Custom hooks (`useApiData`)
A function starting with `use` that uses other hooks, so components can share logic. `useApiData(path)` does "fetch this API path, cancel if the path changes, give me data / error / loading / retry". Product page, homepage blocks, carousel and rows all use it instead of repeating the same 20 lines.

### React Context (the cart)
Context shares a value with every component inside its Provider, without passing props through each level ("prop drilling").
```tsx
<CartProvider>   {/* main.tsx: owns the cart state */}
  <App />        {/* Header (badge), ProductPage (add), CartPage... all call useCart() */}
</CartProvider>
```
`useCart()` reads it. If you forget the Provider, `useCart` throws a clear error.

### localStorage
Key-value storage in the browser that survives refresh and restart, per website. Only strings, so we `JSON.stringify` on save and `JSON.parse` on load (`CartContext.tsx`). It can throw (private mode, full) and can hold **anything** (users can edit it in DevTools), so loading checks every line and drops broken ones. The browser test puts garbage in it and checks the badge.
Only offer ids and quantities are stored, **never prices**: those always come from the server.

### Immutable updates
`setLines(current => current.map(...))` creates a **new** array. React compares old and new state by reference; mutating the old array (`lines.push(x)`) looks like "no change" and the screen doesn't update. Spread `{ ...line, quantity: 2 }` copies an object with one field changed.

### Lazy initial state
`useState(loadCart)` (the function, not `loadCart()`): React calls it only on the first render, so localStorage isn't read on every render.

### The "storage" event
Fires in **other** tabs when localStorage changes. We listen for it, so adding to the cart in one tab updates the badge in another.

### setInterval + cleanup (carousel)
`setInterval` repeats every 6 s; the effect's cleanup `clearInterval` stops it when the component unmounts or when it's paused. Without cleanup, every re-run would start another interval and the slides would speed up.

### Promise in parallel
The product page calls `useApiData` twice, so details and offers load **at the same time**, not one after the other.

### Only follow links/paths you trust
Homepage blocks come from the database and tell the browser which endpoint to call and where "Show all" leads. `isSafeApiPath` only allows `/api/...`, and links must start with `/` but not `//` (`//evil.com` is a full URL to another site). A wrong DB row can't send users or requests to other websites.

### Accessibility in the carousel
`aria-hidden` on invisible slides, `tabIndex={-1}` on their buttons (Tab can't land on something invisible), pause on hover/focus, no auto-play with `prefers-reduced-motion`, labelled arrows and dots.
