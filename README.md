# Eneba-style Marketplace

A digital game key marketplace inspired by [eneba.com](https://www.eneba.com), built as the homework
assignment for the Eneba Software Engineering Internship.

Buyers can browse a catalog, search with typo tolerance, compare offers from several sellers,
buy a game through a simulated checkout and receive a digital key.

| Part | Tech | Hosting |
|---|---|---|
| Frontend | React + TypeScript (Vite) | Vercel |
| Backend | Go, REST API (standard library `net/http`) | Render |
| Database | SQLite, seeded with sample data on startup | inside the backend |

```
/backend    Go API
/frontend   React app
/docs       ARCHITECTURE.md (design + trade-offs), LEARNING_NOTES.md
```

Full design, data model, API list and trade-offs: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Run the backend locally

Needs Go 1.22+ and a C compiler (gcc), because the SQLite driver includes SQLite's C code.

```bash
cd backend
go run .                 # starts on http://localhost:8080 with a freshly seeded database
go test ./...            # run the tests
```

Try it:
```bash
curl http://localhost:8080/api/health
curl http://localhost:8080/api/products
```

Environment variables (all optional locally):

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP port (Render sets it automatically) |
| `ALLOWED_ORIGIN` | `http://localhost:5173` | Frontend URL(s) allowed by CORS, comma-separated |
| `JWT_SECRET` | – | Secret for signing login tokens (used from Day 5) |
| `DB_PATH` | `data.db` | SQLite file; deleted and re-seeded on every start |

## Run the frontend locally

Needs Node 20.19+.

```bash
cd frontend
npm install
npm run dev              # http://localhost:5173, talks to http://localhost:8080 by default
```

To use another API, create `frontend/.env.local` with `VITE_API_URL=https://your-api.example.com`
(see `frontend/.env.example`).

> Live links and screenshots will be added on Day 7.
