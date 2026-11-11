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

> Setup instructions, live links and screenshots will be added on Day 7.
