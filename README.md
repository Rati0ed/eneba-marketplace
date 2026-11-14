# GameKey: an Eneba-style digital games marketplace

Homework assignment for the **Eneba Software Engineering Internship** by Yusif (3rd-year Information Systems & Cybersecurity student, Vilnius University).

Buyers browse a catalog of game keys and gift cards, find games even with typos, compare offers from several sellers, pay with a simulated card and receive their keys instantly.

| | |
|---|---|
| **Live site** | https://eneba-marketplace-zeta.vercel.app |
| **API** | https://eneba-marketplace-api-cbx5.onrender.com/api/health |
| **Demo login** | `demo@gamekey.test` / `gamekey-demo` (has a finished order with keys) |
| **Test cards** | `4242 4242 4242 4242` → payment succeeds · `4000 0000 0000 0002` → declined. Any name, a future expiry (e.g. `12/39`), any 3-digit CVC |

> **First load can take up to a minute.** The API runs on Render's free tier, which sleeps after ~15 minutes without traffic. If a page says "Cannot reach the server", wait a moment and click *Try again*.
>
> **This is a demo store.** No real money is charged, and the keys are generated: they can't be redeemed on Steam, PlayStation, Xbox or Nintendo. Product names belong to their owners; covers are drawn by our API. Not affiliated with Eneba.

---

## What to try (5 minutes)

1. **Search with typos** in the header: `wticher`, `cyberpnk`, `gta 5`, `spiderman`. Then `gpa` + Enter → *Did you mean Grand Theft Auto V?*
2. **Store**: tick *Steam* and *Europe*, sort by price, refresh the page. The filters stay (they live in the URL).
3. **Product page** (e.g. *Elden Ring*): compare sellers. The cheapest in-stock offer is marked *Best price*; sold-out offers are shown but disabled.
4. **Buy something**: add to cart → *Go to checkout* → sign up (or use the demo login) → pay with `…0002` (declined), then `4242…` (succeeds) → your keys appear, hidden until *Reveal*, with *Copy*.
5. **Order history**: *Orders* in the header. The demo account already has one finished order.

## Features

The assignment says "a few features done extensively beat many minimal ones", so three areas got most of the depth:

**1. Typo-tolerant search** (`backend/search/`, own code, no search engine)
- Exact > prefix > 1–2 typos (Damerau-Levenshtein: a swap of two letters counts as one typo) > half-typed word with a typo. Allowed typos grow with word length; words of 1–3 letters must be exact, so `cod` doesn't match `god`.
- Understands names without spaces/apostrophes (`spiderman`, `baldurs`), Roman numerals (`gta 5` → *V*), accents (`pokemon`) and gamer nicknames (`gta`, `rdr2`, `bg3`, `ff7`, …).
- **"Did you mean"** when nothing matches: more forgiving (typos in short words, nicknames with a typo, words typed without vowels like `mncrft`).
- Header dropdown on every page: 250 ms debounce, ↑/↓/Enter/Esc, highlighted matches, "See all N results". Enter opens the store with `?q=`, combinable with all filters.
- Tested with a table of 35 real typos plus "must not match" cases.

**2. Checkout that can't sell a key twice** (`backend/handlers/orders.go`)
- The server prices every order from the database; prices sent by the browser are ignored (tested with a tampered request).
- Payment claims keys, links them to the order and marks it paid in **one transaction**, protected twice: SQLite's write lock is taken at `BEGIN` (`_txlock=immediate`), and keys are claimed with a conditional `UPDATE … WHERE status = 'available'`.
- **Race test**: 8 buyers pay for the last key at the same instant → exactly one succeeds, seven get "sold out". A mutation test confirmed the test fails if the protection is removed.
- Paying twice (or double-clicking) does nothing. A declined card (402) keeps the order open for a retry. Other users' orders return 404.

**3. Store catalog** (`backend/handlers/products.go`)
- Platform / region / genre (multi-select), price range, 5 sort orders, pagination: all in SQL with placeholders, with sort orders from an allow-list.
- Every filter is in the URL (back button, refresh and shared links work). Filter options and counts come from the API.
- Loading skeletons, empty and error states, removable filter chips, mobile filter panel.

**Also built properly:** data-driven homepage (blocks, order and endpoints come from the `homepage_blocks` table), product page with all seller offers, accounts (bcrypt, JWT), server-priced cart, order history, Eneba-inspired responsive design.

## Architecture

```
Browser ──▶ Vercel: React + TypeScript (Vite) single-page app
   │
   └── fetch JSON ──▶ Render: Go REST API (net/http) ──▶ SQLite (fresh + seeded on every start)
```

- **Backend**: Go 1.24, standard library router, `mattn/go-sqlite3`, `golang-jwt/jwt`, `x/crypto/bcrypt`. No framework.
- **Frontend**: React 19, React Router 7, plain CSS with variables. No state library: login and cart use React Context.
- **Data**: 80 products (31 real games in several platform/region variants + gift cards), 8 sellers, 2–5 offers each, ~2,000 keys, created on startup.
- **Auth**: bcrypt-hashed passwords, 24 h HS256 JWTs sent as `Authorization: Bearer …`, middleware that checks signature, algorithm, issuer, expiry and that the user still exists.

Full design, data model, API list and reasoning: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

```
backend/    main.go, db/ (schema, seed, demo account), handlers/ (one file per area), search/, middleware/
frontend/   src/pages, src/components, src/context (auth, cart), src/hooks, src/api.ts
e2e/        smoke.cjs: browser test of the main journeys
docs/       ARCHITECTURE.md, LEARNING_NOTES.md, REVIEW_QUESTIONS.md
```

## Run locally

Needs Go 1.24+ with a C compiler (gcc, for SQLite) and Node 20.19+.

```bash
# API on http://localhost:8080
cd backend
go run .

# Web app on http://localhost:5173 (talks to localhost:8080 by default)
cd frontend
npm install
npm run dev
```

| Backend env variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port (Render sets it) |
| `ALLOWED_ORIGIN` | `http://localhost:5173` | Frontend URL(s) allowed by CORS, comma-separated |
| `JWT_SECRET` | random per run | Signs login tokens. **Set a long random value in production** |
| `DB_PATH` | `data.db` | SQLite file; deleted and re-seeded on every start |

Frontend: `VITE_API_URL` (default `http://localhost:8080`), see `frontend/.env.example`.

## Tests

```bash
cd backend && go test ./...         # 42 Go tests: search, filters, auth, cart, checkout, races
cd e2e && npm install && npx playwright install chromium
node smoke.cjs                      # 12 browser checks against localhost (backend + frontend running)
BASE_URL=https://eneba-marketplace-zeta.vercel.app node smoke.cjs   # same, against the live site
```

Highlights: typo table, SQL-injection attempt in `sort`, forged / expired / `alg: none` JWTs, tampered cart prices, double payment, 8-buyer last-key race, private orders, Luhn and expiry checks, demo account seed.

## Trade-offs (on purpose)

| Choice | Why |
|---|---|
| SQLite, recreated on every start | Zero setup and identical demo data each time. Render's free tier has no persistent disk anyway. |
| Own search instead of Elasticsearch/Meilisearch | 80 products fit in memory; small, tested, fully explainable code. |
| Simulated payments | The assignment says no real money; the flow (pending → paid / declined / failed) is still realistic. |
| Cart in the browser | Guests can shop before signing up; the server re-prices everything. |
| JWT in `localStorage` | Frontend and API are on different domains, where cookies get blocked. Risk: XSS could read it. Mitigated by React's escaping, no raw HTML, 24 h expiry. |
| Keys claimed at payment, not reserved | Simpler. A buyer can lose the last key between order and payment; they get a clear "sold out" and aren't charged. |
| Generated SVG covers | No rights to publisher art, and no external image host that could break. |
| Few dependencies | Every line should be explainable in an interview. |

## Known limitations

- **Render sleep + database reset**: after ~15 min idle the API sleeps; the next request wakes it (up to ~1 min) and **starts from a fresh database**. Accounts and orders made earlier are gone (the demo account is always recreated). A real deployment would use a managed database.
- No login rate limiting, password reset or email verification.
- JWTs can't be revoked before they expire (24 h).
- No admin panel: catalog and homepage blocks are changed in the seed data.
- `golang.org/x/crypto` is pinned to v0.48.0, the newest version that built with Go 1.24 in my build environment.

## How I used AI

Eneba encourages AI use and will read the full conversation, so here is exactly how it was used:

- **A coding agent (Claude, in one long chat)** wrote the code step by step, ran the tests and browser checks, committed and pushed. At every step it explained what it built in plain language (compared to Python where useful), kept [docs/LEARNING_NOTES.md](docs/LEARNING_NOTES.md) with every new concept, and added questions about each part to [docs/REVIEW_QUESTIONS.md](docs/REVIEW_QUESTIONS.md).
- **A separate AI assistant** helped me plan the steps and write the prompts for the coding agent.
- **What I did myself:**
  - Chose the stack and the scope.
  - Answered the agent's check questions.
  - Deployed to Render and Vercel.
  - Tested the live site step by step and reported bugs back. Two of my findings led to fixes: "Did you mean" was weak for short queries like `gpa`, and the demo keys needed a clear "not redeemable" note.
  - Before submitting, I go through the code using the review questions, in a review session with the agent.
- My background is Python and some SQL; Go, React and TypeScript were new to me. The learning notes and review questions are how I made sure I can explain every part of the code.
