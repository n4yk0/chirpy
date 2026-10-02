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

Everything is JSON over HTTP, served from `http://localhost:1337`. Errors come
back as `{"error": "<message>"}` with the matching status code.

### Authentication

Two tokens, two jobs:

| | Access token | Refresh token |
| --- | --- | --- |
| Format | JWT, HS256 | 256 bits of randomness, hex |
| Lifetime | 1 hour | 60 days |
| Stored server-side | no | yes, in `refresh_tokens` |
| Revocable | no | yes, via `POST /api/revoke` |
| Used on | every authenticated endpoint | `/api/refresh` and `/api/revoke` only |

Both travel in the same header, `Authorization: Bearer <token>`. The access
token is deliberately unrevocable: nothing is looked up to validate it, only
the signature is checked, which is what makes it cheap. The refresh token is
the part you can take back, so it is the only one worth storing.

The Polka webhook uses a third scheme, a shared API key:
`Authorization: ApiKey <POLKA_KEY>`.

### Users and sessions

| Method | Path | Auth | Returns |
| --- | --- | --- | --- |
| `POST` | `/api/users` | — | `201` with the user |
| `POST` | `/api/login` | — | `200` with the user, an access token and a refresh token |
| `PUT` | `/api/users` | access token | `200` with the updated user |
| `POST` | `/api/refresh` | refresh token | `200` with a fresh access token |
| `POST` | `/api/revoke` | refresh token | `204`, the refresh token is dead |

`POST /api/users` and `PUT /api/users` both take `{"email", "password"}`.
Passwords are hashed with argon2id and never returned.

### Chirps

| Method | Path | Auth | Returns |
| --- | --- | --- | --- |
| `POST` | `/api/chirps` | access token | `201` with the chirp |
| `GET` | `/api/chirps` | — | `200` with every chirp |
| `GET` | `/api/chirps/{chirpID}` | — | `200`, or `404` if unknown |
| `DELETE` | `/api/chirps/{chirpID}` | access token | `204`, or `403` if you are not the author |

`GET /api/chirps` takes two optional query parameters:

| Parameter | Values | Effect |
| --- | --- | --- |
| `author_id` | a user UUID | Only that author's chirps; `400` if it is not a UUID |
| `sort` | `asc` (default), `desc` | Order by creation date |

Chirps are capped at 140 characters, and a small profanity list is masked
before storage.

### Admin and webhooks

| Method | Path | Auth | Returns |
| --- | --- | --- | --- |
| `GET` | `/api/healthz` | — | `200 OK`, plain text |
| `GET` | `/admin/metrics` | — | HTML page with the file-server hit count |
| `POST` | `/admin/reset` | `PLATFORM=dev` | `200` after deleting every user, `403` otherwise |
| `POST` | `/api/polka/webhooks` | API key | `204` |
| `GET` | `/app/` | — | Static file server, counted by the metrics middleware |

`POST /admin/reset` is destructive and refuses to run outside a dev platform.

The webhook body is `{"event": "...", "data": {"user_id": "..."}}`. Only
`user.upgraded` does anything — it flips `is_chirpy_red` on the user. Any
other event is acknowledged with `204` and ignored, because a sender that
gets an error will keep retrying. A wrong or missing API key is `401`, an
unknown user is `404`.

### A full exchange

```bash
curl -X POST localhost:1337/api/users \
  -d '{"email":"user@example.com","password":"correct horse battery staple"}'

TOKEN=$(curl -sX POST localhost:1337/api/login \
  -d '{"email":"user@example.com","password":"correct horse battery staple"}' \
  | jq -r .token)

curl -X POST localhost:1337/api/chirps \
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
POLKA_KEY="the-key-polka-signs-its-webhooks-with"
PLATFORM="dev"
```

Generate `JWT_SECRET` with `openssl rand -base64 64`.

Apply the migrations, then start the server:

```bash
goose -dir sql/schema postgres "$DB_URL" up
go run .
```

The server listens on port `1337`.

Regenerate the database layer after editing anything under `sql/` with
`sqlc generate`.

## Tests

```bash
go test ./...
```

The `internal/auth` suite covers password hashing and verification, salt
randomness, the JWT round-trip, expired tokens, a wrong signing secret, an
unexpected signing method, malformed tokens, and the parsing of both
`Authorization` schemes.

## Progress

Boot.dev *Learn Web Servers*, complete through the Webhooks chapter:

- [x] Static file server, metrics middleware, readiness endpoint
- [x] PostgreSQL, goose migrations, sqlc-generated queries
- [x] Users, argon2id password hashing
- [x] Chirps: create, list, fetch by id, validation and profanity masking
- [x] Login with a signed JWT
- [x] Refresh tokens, with revocation
- [x] Authorization: users update their own account, authors delete their own chirps
- [x] Polka webhooks, authenticated with an API key
- [x] This README

## About

I'm Alexis, a fullstack software engineer at an IT consultancy in France. I
work mostly with Next.js, React, Laravel and Django. Go is what I'm sharpening
at the moment, and this repository is part of that.

Background and CV: **[nayko.dev](https://nayko.dev)**

## Credits

Project brief and test suite from the [Boot.dev](https://boot.dev) *Learn Web
Servers* course. The implementation is my own.
