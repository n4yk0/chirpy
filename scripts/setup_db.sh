#!/usr/bin/env bash
# One-time database setup for Chirpy on Ubuntu 24.04 / WSL2.
# Installs PostgreSQL 16, sets the postgres role password, creates the
# `chirpy` database, and applies the goose migrations.
#
# Run from anywhere:  bash scripts/setup_db.sh
set -euo pipefail

# Move to the repo root (parent of this script's dir) so relative paths work.
cd "$(dirname "$0")/.."

echo "==> Installing PostgreSQL 16 (default in Ubuntu 24.04 repos)"
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib

echo "==> PostgreSQL version:"
psql --version

echo "==> Starting the PostgreSQL server (WSL uses service, not systemd)"
sudo service postgresql start

echo "==> Setting a password for the 'postgres' role"
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';"

echo "==> Creating the 'chirpy' database (if it does not already exist)"
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='chirpy'" | grep -q 1; then
  sudo -u postgres createdb chirpy
  echo "    created database 'chirpy'"
else
  echo "    database 'chirpy' already exists"
fi

echo "==> Running goose migrations"
export PATH="$PATH:$(go env GOPATH)/bin"
goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" up

echo "==> Done. Start the server with:  go run ."
