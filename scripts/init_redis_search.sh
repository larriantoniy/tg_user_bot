#!/bin/bash
set -euo pipefail

echo "Creating RediSearch index 'idx:predictions'…"

# Ждём, пока Redis точно поднимется
until redis-cli -h 127.0.0.1 -p 6379 ping; do
  echo "Waiting for Redis…"
  sleep 1
done

# Пытаемся создать индекс
if redis-cli -h 127.0.0.1 -p 6379 FT.CREATE idx:predictions ON JSON PREFIX 1 prediction: SCHEMA $.RawText AS rawText TEXT; then
  echo "Index 'idx:predictions' created successfully."
else
  echo "Index 'idx:predictions' already exists or failed to create."
fi


echo "Verifying created indexes:"
redis-cli -h 127.0.0.1 -p 6379 FT._LIST