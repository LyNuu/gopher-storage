#!/bin/sh
set -e

if [ -n "$CONNECTION_STRING" ]; then
  echo "[entrypoint] running migrations..."
  goose -dir /app/migrations postgres "$CONNECTION_STRING" up
fi

echo "[entrypoint] starting service..."
exec /app/app
