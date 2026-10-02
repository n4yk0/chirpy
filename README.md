# Chirpy

A Twitter-like HTTP API in Go, backed by PostgreSQL. Users register, log in,
and post short messages ("chirps"). Built from scratch on the standard
library as part of the [Boot.dev](https://boot.dev) DevOps path, in the
*Learn Web Servers* course: no web framework, no ORM.

## What it does

- Registration and login with email and password
- Post, list, read and delete chirps, capped at 140 characters
- Sessions built on a short-lived access token and a long-lived, revocable
  refresh token
- Users update their own account; authors delete their own chirps
- Paid "Chirpy Red" upgrades, applied from a payment provider's webhook
- Static file server with a request counter, and a dev-only reset endpoint

## What it demonstrates

- **HTTP without a framework** — method-aware routing, path wildcards and
  handler-wrapping middleware on `net/http` alone
- **Authentication designed, not copied** — HS256 JWTs validated against
  expiry *and* signing method, refresh tokens stored and revocable,
  argon2id password hashing, two `Authorization` schemes over three
  distinct credentials
- **Authorization as a distinct concern** — 401 for who you are, 403 for
  what you may touch
- **SQL kept first-class** — versioned goose migrations, queries written as
  SQL and compiled to type-safe Go by sqlc
- **A webhook receiver that survives retries** — unknown events are
  acknowledged rather than rejected, since a sender that gets an error
  replays it
- **Table-driven tests on the security-critical package** — hashing, token
  round-trips, expiry, wrong secret, forged signing method, malformed input
- **The details REST gets wrong** — accurate status codes, optional query
  parameters that default instead of failing, and errors that never reveal
  which half of a credential was wrong

## Stack

| Concern | Choice |
| --- | --- |
| HTTP | `net/http` `ServeMux` (Go 1.22+ method and wildcard routing) |
| Database | PostgreSQL |
| Migrations | [goose](https://github.com/pressly/goose) |
| Data access | [sqlc](https://sqlc.dev) — type-safe Go generated from plain SQL |
| Password hashing | [argon2id](https://github.com/alexedwards/argon2id) |
| Tokens | [golang-jwt/jwt v5](https://github.com/golang-jwt/jwt), HS256 |

## Install and run

Requirements: Go 1.26+, PostgreSQL, [goose](https://github.com/pressly/goose)
and, to regenerate the data layer, [sqlc](https://sqlc.dev).

```bash
git clone https://github.com/n4yk0/chirpy.git
cd chirpy
go mod download
```

Create a database, then a `.env` file at the repository root (it is
git-ignored):

```
DB_URL="postgres://user:password@localhost:5432/chirpy?sslmode=disable"
JWT_SECRET="a-long-random-string"
POLKA_KEY="the-key-the-payment-provider-signs-its-webhooks-with"
PLATFORM="dev"
```

Generate `JWT_SECRET` with `openssl rand -base64 64`. `PLATFORM` must be
`dev` for `POST /admin/reset` to work, and anything else in production.

Apply the migrations and start the server on port `1337`:

```bash
goose -dir sql/schema postgres "$DB_URL" up
go run .
```

Run `sqlc generate` after editing anything under `sql/`.

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

Both travel as `Authorization: Bearer <token>`. The access token is
unrevocable by design — validating it reads nothing but its own signature,
which is what makes it cheap — so the refresh token is the one worth
storing. The webhook endpoint uses a third scheme,
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
`user.upgraded` acts, flipping `is_chirpy_red`; every other event is
acknowledged and ignored, because a sender that gets an error replays it.
A wrong or missing key is `401`, an unknown user `404`.

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

## Tests

```bash
go test ./...
```

The `internal/auth` suite covers password hashing and verification, salt
randomness, the JWT round-trip, expired tokens, a wrong signing secret, an
unexpected signing method, malformed tokens, and the parsing of both
`Authorization` schemes.

## About

I'm Alexis, a fullstack software engineer at an IT consultancy in France. I
work mostly with Next.js, React, Laravel and Django. Go is what I'm sharpening
at the moment, and this repository is part of that.

Background and CV: **[nayko.dev](https://nayko.dev)**

## Credits

Project brief and test suite from the [Boot.dev](https://boot.dev) *Learn Web
Servers* course. The implementation is my own.
