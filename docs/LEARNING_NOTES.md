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
