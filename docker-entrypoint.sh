#!/bin/sh
set -e

# Build DATABASE_URL from individual components
if [ -n "$DB_HOST" ]; then
    DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
    
    echo "Running database migrations..."
    for migration in /migrations/*.sql; do
        echo "Applying: $migration"
        psql "$DATABASE_URL" -f "$migration" || true
    done
    echo "Migrations complete."
fi

exec "$@"

