# scripts/init_redis_search.sh
redis-cli <<EOF
# индексируем JSON-документы под префиксом prediction:
FT.CREATE idx:predictions ON JSON
  PREFIX 1 prediction:
  SCHEMA $.RawText AS rawText TEXT
EOF
