#!/usr/bin/env bash
# Colorized Docker Compose logs for live-pulse
# Usage: ./logs.sh [lines]  (default: 50 tail lines)

LINES=${1:-50}

docker compose -f docker-compose.prod.yml logs --tail="$LINES" --follow \
  | sed -E \
    -e "s/^(livepulse-api\s*\|)/\x1b[1;32m\1\x1b[0m/"    \
    -e "s/^(livepulse-web\s*\|)/\x1b[1;34m\1\x1b[0m/"     \
    -e "s/^(livepulse-realtime\s*\|)/\x1b[1;35m\1\x1b[0m/" \
    -e "s/(\"status\":\s*[45][0-9]{2})/\x1b[1;31m\1\x1b[0m/g" \
    -e "s/(ERR|ERROR|FATAL|panic)/\x1b[1;31m\1\x1b[0m/gI"     \
    -e "s/(WARN|WARNING)/\x1b[1;33m\1\x1b[0m/gI"              \
    -e "s/(\"status\":\s*2[0-9]{2})/\x1b[32m\1\x1b[0m/g"
