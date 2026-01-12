#!/bin/sh
set -e

if [ -z "$DATABASE_URL" ] && [ -n "$DB_HOST" ]; then
    DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
fi

# Run migrations if we have a database URL
if [ -n "$DATABASE_URL" ]; then
    echo "Running database migrations..."
    for migration in /migrations/*.sql; do
        echo "Applying: $migration"
        psql "$DATABASE_URL" -f "$migration" || true
    done
    echo "Migrations complete."
fi

exec "$@"

