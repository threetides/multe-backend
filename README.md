# multe_backend

The backend of multe, the threetides studio's app for personal collections,
served at [api.multe.app](https://api.multe.app).

multe lets you build collections of the things you own: discs, hiking gear,
gym equipment, books. Custom fields are defined per collection, so every item
in it shares the same fields (speed/glide/turn/fade for discs, weight and
category for hiking gear) on top of standard ones like price, location, and
date bought. Each collection tracks totals such as its number of items and
total price, kept visibly distinct for owned and wishlisted items. Items in
any collection can be wishlisted: a wishlisted item shows that state in its
collection and appears on a global wishlist across all collections, and when
bought it transitions to owned in place (filling in price and date bought at
that moment) rather than being re-added. The core of the app is item
management and wishlists; further out, wishlists may become shareable with
others (for birthdays and christmases) and the web app may be joined by a
native one.

## Development

`multe_backend` is a Go HTTP service built on the standard library's
`net/http`, backed by Postgres. Configuration is read from a `.env` file (or
the environment): `DB_CONNECTION_STRING`, `PORT`, `CORS_ORIGIN`,
`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and `REDIRECT_URL`.

With Go, Postgres, and [golangci-lint](https://golangci-lint.run) installed:

```sh
make dev    # lint, then run the server
```

## Project structure

```
multe_backend/
├── main.go           wiring: database, CORS, routes
├── internal/
│   ├── auth/         Google sign-in, sessions, and auth middleware
│   ├── httpx/        HTTP client and JSON response helpers
│   ├── sql/          the database schema (db.sql)
│   └── test/         a protected test route
└── makefile
```

## License

See [LICENSE](LICENSE).
