#!/usr/bin/env bash
set -uo pipefail

BOLD=$'\033[1m'; YELLOW=$'\033[33m'; NC=$'\033[0m'

echo "${BOLD}Métricas Prometheus — FIAP X${NC}"
echo "Dica: rode 'make up' antes. Com a stack no ar, são exibidos os valores;"
echo "sem ela, ficam apenas as URLs dos endpoints."
echo

show() {
  local name="$1" port="$2" filter="$3"
  local url="http://localhost:${port}/metrics"
  printf '  %s%-20s%s  %s\n' "$BOLD" "$name" "$NC" "$url"
  if out="$(curl -fsS --max-time 3 "$url" 2>/dev/null)"; then
    printf '%s' "$out" | grep -E "$filter" | sed 's/^/      /'
  else
    printf '      %s! serviço indisponível (endpoint acima)%s\n' "$YELLOW" "$NC"
  fi
  echo
}

show "auth-service"        8081 'auth_(registrations|logins)_total'
show "video-api"           8080 'videos_(uploaded|downloaded)_total'
show "processing-worker"   9092 'videos_processed_total|video_processing_errors_total|video_processing_duration_seconds_count'
show "notification-service" 9093 'emails_(sent|failed)_total'

echo "${BOLD}Outras métricas (HTTP):${NC}"
echo "  http_requests_total, http_request_duration_seconds — auth-service (:8081) e video-api (:8080)"
