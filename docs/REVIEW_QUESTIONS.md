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

---

## Day 2: Store (backend)
Files: `backend/handlers/products.go`, `backend/handlers/cards.go`, `backend/handlers/products_test.go`

1. **`products.go`**: why can't `sort` be passed into SQL with a `?` placeholder, and how does `sortOptions` keep us safe from SQL injection anyway?
2. **`products.go`**: what does `?platform=Steam&platform=Xbox&region=Europe` return? Which conditions are joined with OR and which with AND?
3. **`cards.go`**: why do we need the `WITH cards AS (...)` CTE to filter by lowest price? Why doesn't `WHERE MIN(o.price_cents) <= ?` work?
4. **`products.go`**: why are prices in the URL whole euros while everything else uses cents? Where exactly do we convert?
5. **`products.go`**: a request for `page=99` returns 200 with no items, but `page=0` returns 400. Why the difference?

## Day 2: Store (frontend)
Files: `frontend/src/pages/StorePage.tsx`, `frontend/src/components/FilterSidebar.tsx`, `frontend/src/components/ProductCard.tsx`, `frontend/vercel.json`

1. **`StorePage.tsx`**: the filters are stored only in the URL. Name three things that work *because* of that.
2. **`StorePage.tsx`**: there's no `loading` state variable. How does the page know it's loading?
3. **`StorePage.tsx`**: what bug would appear without `controller.abort()` in the effect cleanup if the user clicks two filters quickly?
4. **`FilterSidebar.tsx`**: why do the price inputs only update the URL on "Apply", and what does `key={`${minPrice}-${maxPrice}`}` do?
5. **`vercel.json`**: what happens on the live site if you refresh `/store?platform=Steam` without the rewrite rule?

---

## Day 3: Search (backend)
Files: `backend/search/search.go`, `backend/search/words.go`, `backend/search/search_test.go`, `backend/handlers/search.go`, `backend/handlers/products.go`

1. **`search.go`**: what is the edit distance between `wticher` and `witcher` in our algorithm, and why is it different from plain Levenshtein?
2. **`search.go` → `allowedTypos`**: why does a 3-letter word get 0 allowed typos? Give an example of what would go wrong with 1.
3. **`words.go`**: how does `spiderman` find "Marvel's Spider-Man 2", and how does `gta 5` find "Grand Theft Auto V"? (Two different mechanisms.)
4. **`search.go`**: why do we drop results that score under half of the best one? Which test shows the problem it fixes?
5. **`handlers/products.go`**: with `?q=elden&platform=PlayStation&sort=price_asc`, which part does the search index do, and which part does SQL do?
6. **`search.go`**: why do we use `[]rune(word)` instead of `len(word)` when measuring words?

## Day 3: Search (frontend)
Files: `frontend/src/components/SearchBar.tsx`, `frontend/src/components/Highlight.tsx`, `frontend/src/components/Header.tsx`

1. **`SearchBar.tsx`**: explain how the debounce works using only `setTimeout`, `clearTimeout` and the effect cleanup. How many requests does typing "elden" quickly send?
2. **`SearchBar.tsx`**: why does the dropdown have `onMouseDown={(e) => e.preventDefault()}`? What breaks without it?
3. **`SearchBar.tsx`**: what happens on Enter (a) with a result highlighted, (b) with nothing highlighted?
4. **`Highlight.tsx`**: what would happen if a user typed `c++ (pc` and we didn't escape the regex characters?
5. **`Header.tsx`**: what does `key={urlQuery}` on `<SearchBar>` achieve when the user presses the browser's back button?

---

## Day 4: Product page and homepage (backend)
Files: `backend/handlers/product.go`, `backend/handlers/home.go`, `backend/handlers/routes.go`, `backend/handlers/home_test.go`, `backend/db/schema.sql`

1. **`routes.go`**: both `/api/products/filters` and `/api/products/{slug}` match the URL `/api/products/filters`. Which handler runs, and why?
2. **`product.go`**: how are the offers sorted, and why is a cheaper sold-out offer shown after a more expensive in-stock one?
3. **`home.go` + `home_test.go`**: how would you add a fourth homepage row "Nintendo deals" without touching the frontend? What would you need to add in the backend?
4. **`product.go`**: why does a missing product return 404 and not 500? Which Go error tells us it's missing?

## Day 4: Product page and homepage (frontend)
Files: `frontend/src/hooks/useApiData.ts`, `frontend/src/context/CartContext.tsx`, `frontend/src/pages/HomePage.tsx`, `frontend/src/components/BannerCarousel.tsx`, `frontend/src/pages/ProductPage.tsx`, `frontend/src/api.ts`

1. **`HomePage.tsx`**: walk through what happens from opening `/` to seeing the "Top deals" row. How many API requests are made, and which part decides their order?
2. **`api.ts` → `isSafeApiPath`**: what attack or mistake does it prevent? Give an example of a value it rejects.
3. **`CartContext.tsx`**: why does the cart store only `offer_id` and `quantity`, and not the price? What happens if a user edits localStorage to `quantity: 999`?
4. **`CartContext.tsx`**: why do we write `current.map(...)` / `[...current, x]` instead of changing the array directly?
5. **`BannerCarousel.tsx`**: what would go wrong without `return () => clearInterval(timer)`?
6. **`useApiData.ts`**: why does it return `data: null` while a new path is loading, instead of the previous data?

---

## Day 5: Accounts and cart (backend)
Files: `backend/handlers/auth.go`, `backend/handlers/auth_test.go`, `backend/handlers/cart.go`, `backend/main.go`

1. **`auth.go`**: why bcrypt and not SHA-256 for passwords? What are the "salt" and the "cost"?
2. **`auth.go`**: why do we reject passwords longer than 72 bytes?
3. **`auth.go` → `Login`**: why is `bcrypt.CompareHashAndPassword(dummyHash, ...)` called when the email doesn't exist?
4. **`auth.go` → `parseToken`**: what is the `"alg": "none"` attack, and which line stops it?
5. **`auth.go` → `RequireAuth`**: after checking the signature, why do we still look the user up in the database?
6. **`cart.go`**: a user sends `{"offer_id": 5, "quantity": 1, "unit_price_cents": 1}`. Where does the price used by the server come from, and why is `unit_price_cents` ignored?
7. **`main.go`**: what happens if `JWT_SECRET` is not set? Why is that acceptable locally but would be bad with a persistent database?

## Day 5: Accounts and cart (frontend)
Files: `frontend/src/context/AuthContext.tsx`, `frontend/src/components/RequireAuth.tsx`, `frontend/src/hooks/useAuth.ts`, `frontend/src/pages/AuthPage.tsx`, `frontend/src/pages/CartPage.tsx`

1. **`AuthContext.tsx`**: walk through what happens when a signed-in user presses F5. Which request is made, and what if it returns 401?
2. **`RequireAuth.tsx`**: if someone removes `<RequireAuth>` with browser dev tools, can they see other people's orders? Why not?
3. **`useAuth.ts` → `safeNextPath`**: what is an open redirect? Give a URL it blocks.
4. **`AuthContext.tsx`**: what's the security trade-off of keeping the JWT in localStorage instead of a cookie, and why did we accept it?

## Day 6: Checkout (backend)
Files: `backend/handlers/orders.go`, `backend/handlers/orders_test.go`, `backend/db/db.go`

1. **`orders.go` → `completeOrder`**: list the steps inside the transaction. What could go wrong if they weren't in one transaction?
2. **`orders.go` + `db.go`**: name the TWO protections that stop a key from being sold twice. What did the mutation test show when we removed `_txlock=immediate`?
3. **`orders_test.go` → `TestLastKeyRace`**: why was the first version of this test useless, and what does `testHookBeforeClaim` change?
4. **`orders.go` → `PayOrder`**: what happens on a second payment of a paid order? What's the word for this property?
5. **`orders.go`**: why does a declined card return 402 and leave the order `pending`, while "sold out" returns 409 and marks it `failed`?
6. **`orders.go` → `loadOrder`**: user B asks for user A's order 7. What does B get, and why not 403 Forbidden?
7. **`orders.go`**: what do we store about the card, and why only that?

## Day 6: Checkout (frontend)
Files: `frontend/src/pages/CheckoutPage.tsx`, `frontend/src/pages/OrderPage.tsx`, `frontend/src/components/KeyBox.tsx`

1. **`CheckoutPage.tsx`**: the first card is declined, the second works. How many orders are created, and which variable makes that happen?
2. **`CheckoutPage.tsx`**: explain the bug where buyers landed on `/cart` after paying, and how `paidOrderId` fixes it.
3. **`CheckoutPage.tsx`**: what are the two layers that stop a double-click from paying twice?
4. **`KeyBox.tsx`**: why are keys hidden by default, and what happens if the clipboard is blocked?
