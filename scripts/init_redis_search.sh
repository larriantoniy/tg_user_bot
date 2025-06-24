# scripts/init_redis_search.sh
set -euo pipefail

echo "Creating RediSearch index 'idx:predictions'…"
# Пытаемся создать индекс; если он уже есть — сообщаем об этом
if redis-cli FT.CREATE idx:predictions ON JSON PREFIX 1 prediction: SCHEMA $.RawText AS rawText TEXT; then
  echo "Index 'idx:predictions' created successfully."
else
  echo "Index 'idx:predictions' already exists or failed to create."
fi

echo "Verifying created indexes:"
# В выводе должно быть: idx:predictions
redis-cli FT._LIST