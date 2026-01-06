#!/bin/bash
set -e

# Migration script for Render PostgreSQL database
# Usage: ./scripts/migrate.sh

if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable not set"
    echo "For Render, get this from: Dashboard > Database > Connection String (External)"
    exit 1
fi

echo "Running database migrations..."

# Install psql if not available (for local testing)
if ! command -v psql &> /dev/null; then
    echo "psql not found. Please install PostgreSQL client."
    exit 1
fi

# Run all migration files in order
for migration in migrations/*.sql; do
    echo "Applying: $migration"
    psql "$DATABASE_URL" -f "$migration"
done

echo "Migrations completed successfully!"

