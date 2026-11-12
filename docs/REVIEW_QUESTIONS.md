# Review Questions

Questions about what we built, step by step. In the review session, answer each in your own words,
then open the related files and check your answer against the code.

---

## Day 1 · Step 1: Repo skeleton
Files: `.gitignore`, `README.md`

1. ~~Why is `*.db` in `.gitignore`?~~ (answered on Day 1)
2. ~~A secret was pushed to a public repo. Is deleting it in the next commit enough?~~ (answered on Day 1: no, rotate the secret)

---

## Day 1 · Step 2: Backend
Files: `backend/main.go`, `backend/db/db.go`, `backend/db/schema.sql`, `backend/db/seed.go`,
`backend/handlers/products.go`, `backend/handlers/covers.go`, `backend/middleware/middleware.go`

1. **`db/db.go`**: the server deletes the database file and re-seeds it on every start. Give two reasons this is fine for our project, and one reason it would be unacceptable in a real shop.
2. **`db/db.go` + `db/db_test.go`**: why do we turn on foreign keys in the connection string (`_foreign_keys=on`) instead of running `PRAGMA foreign_keys = ON` once after opening? What would go wrong otherwise?
3. **`handlers/products.go`**: why is `LowestPriceCents` a `*int` (pointer) and not an `int`? What does the frontend receive for a product whose offers are all sold out?
4. **`handlers/products.go`**: why does the query use `LEFT JOIN` instead of `JOIN`? Which product would disappear from the list with a plain `JOIN`?
5. **`middleware/middleware.go`**: a hacker runs `curl https://our-api/api/products` from their laptop. Does our CORS setup stop them? What *does* CORS protect against?
6. **`db/seed.go`**: why do we use `rand.New(rand.NewSource(42))` instead of a truly random generator? Would that be OK for real game keys?
7. **`handlers/covers.go`**: what could happen if we put the product title into the SVG without `html.EscapeString`?

---

## Day 1 · Step 3: Frontend
Files: `frontend/src/api.ts`, `frontend/src/App.tsx`, `frontend/src/vite-env.d.ts`, `frontend/.env.example`

1. **`src/api.ts`**: you change `VITE_API_URL` in Vercel's settings, but the live site still calls the old URL. Why, and what do you do?
2. **`src/api.ts`**: someone suggests storing an API key for a payment provider in `VITE_PAYMENT_KEY`. Why is that a bad idea?
3. **`src/App.tsx`**: why is the request inside `useEffect` and not written directly in the component function body? What would happen if it were in the body?
4. **`src/App.tsx`**: what are the three states the page can be in, and what does the user see in each?
5. **`src/api.ts`**: `fetch` gets a `404` response. Does it throw an error? How does `apiGet` handle it?
