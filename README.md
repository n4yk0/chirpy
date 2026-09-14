# Chirpy

A Twitter-like HTTP API written in Go, backed by PostgreSQL. Users register,
log in against argon2id password hashes, receive a signed JWT, and post
short messages ("chirps").

This project is built as part of the [Boot.dev](https://boot.dev) **DevOps
career path**, in the *Learn Web Servers* course. It is a learning project,
written from scratch rather than scaffolded: routing, authentication,
migrations and data access are all hand-rolled on the Go standard library.

## Stack

| Concern | Choice |
| --- | --- |
| HTTP | `net/http` `ServeMux` (Go 1.22+ method and wildcard routing) — no framework |
| Database | PostgreSQL |
| Migrations | [goose](https://github.com/pressly/goose) |
| Data access | [sqlc](https://sqlc.dev) — type-safe Go generated from plain SQL |
| Password hashing | [argon2id](https://github.com/alexedwards/argon2id) |
| Tokens | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt), HS256 |

No ORM and no web framework: the point of the exercise is to understand what
those layers actually do.

## API

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/api/healthz` | — | Readiness probe |
| `POST` | `/api/users` | — | Create a user (email + password) |
| `POST` | `/api/login` | — | Authenticate, returns the user and a JWT |
| `POST` | `/api/chirps` | Bearer JWT | Post a chirp |
| `GET` | `/api/chirps` | — | List all chirps |
| `GET` | `/api/chirps/{chirpID}` | — | Fetch a single chirp |
| `GET` | `/admin/metrics` | — | File-server hit counter |
| `POST` | `/admin/reset` | dev only | Reset hits and delete all users |
| `GET` | `/app/` | — | Static file server (instrumented) |

Chirps are capped at 140 characters and a small profanity list is masked
before storage. `POST /admin/reset` is destructive and refuses to run unless
`PLATFORM=dev`.

### Example

```bash
curl -X POST localhost:8090/api/users \
  -d '{"email":"user@example.com","password":"correct horse battery staple"}'

TOKEN=$(curl -sX POST localhost:8090/api/login \
  -d '{"email":"user@example.com","password":"correct horse battery staple"}' \
  | jq -r .token)

curl -X POST localhost:8090/api/chirps \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"body":"Hello world"}'
```

## Running it

Requirements: Go 1.26+, PostgreSQL, and [goose](https://github.com/pressly/goose)
for the migrations.

Create a `.env` file at the repository root (it is git-ignored):

```
DB_URL="postgres://user:password@localhost:5432/chirpy?sslmode=disable"
JWT_SECRET="a-long-random-string"
PLATFORM="dev"
```

Generate `JWT_SECRET` with `openssl rand -base64 64`.

Apply the migrations, then start the server:

```bash
goose -dir sql/schema postgres "$DB_URL" up
go run .
```

The server listens on port `8090`.

Regenerate the database layer after editing anything under `sql/` with
`sqlc generate`.

## Tests

```bash
go test ./...
```

The `internal/auth` suite covers the hashing round-trip, salt randomness, JWT
round-trip, expired tokens, a wrong signing secret, an unexpected signing
method, and malformed tokens.

## Progress

Chapters completed up to **chapter 6 — Authentication**, currently at lesson 8
(refresh tokens).

- [x] Static file server, metrics middleware, readiness endpoint
- [x] PostgreSQL, goose migrations, sqlc-generated queries
- [x] Users, argon2id password hashing
- [x] Chirps: create, list, fetch by id, validation and profanity masking
- [x] Login with a signed JWT
- [ ] Refresh tokens and short-lived access tokens
- [ ] Authorization on update and delete
- [ ] Webhooks, API keys
- [ ] Deployment

## About

I'm Alexis, a fullstack software engineer at an IT consultancy in France. I
work mostly with Next.js, React, Laravel and Django. Go is what I'm sharpening
at the moment, and this repository is part of that.

Background and CV: **[nayko.dev](https://nayko.dev)**

## Credits

Project brief and test suite from the [Boot.dev](https://boot.dev) *Learn Web
Servers* course. The implementation is my own.
