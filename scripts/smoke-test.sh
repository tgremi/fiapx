#!/usr/bin/env bash
set -euo pipefail

# Smoke test end-to-end do FIAP X:
#   health → register → login → upload → processamento → list → download.
# Pré-requisitos: sistema rodando (`make up`), curl, ffmpeg, python3.

AUTH_URL="${AUTH_URL:-http://localhost:8081}"
VIDEO_URL="${VIDEO_URL:-http://localhost:8080}"
MAILHOG_URL="${MAILHOG_URL:-http://localhost:8025}"
S3_CONTAINER="${S3_CONTAINER:-fiapx-s3}"
TIMEOUT_STATUS="${TIMEOUT_STATUS:-60}"

RED=$'\033[31m'; GREEN=$'\033[32m'; YELLOW=$'\033[33m'; NC=$'\033[0m'

step() { printf '\n\033[1m==> %s\033[0m\n' "$1"; }
ok()   { printf '   %s✓ %s%s\n' "$GREEN" "$1" "$NC"; }
fail() { printf '   %s✗ %s%s\n' "$RED" "$1" "$NC"; exit 1; }
warn() { printf '   %s! %s%s\n' "$YELLOW" "$1" "$NC"; }

get_field() {
  python3 -c 'import sys,json; print(json.loads(sys.argv[1])[sys.argv[2]])' "$1" "$2"
}

http_code() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}

# ---- pré-checks -------------------------------------------------------------
for cmd in curl ffmpeg python3; do
  command -v "$cmd" >/dev/null 2>&1 || fail "comando '$cmd' não encontrado"
done

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

EMAIL="smoke-$(date +%s)@fiapx.com"
PASSWORD="senha123"
VIDEO="$WORK/sample.mp4"
ZIP="$WORK/frames.zip"
BAD_VIDEO="$WORK/bad.mp4"

# ---- 1. health auth ---------------------------------------------------------
step "1. Health do auth-service"
if curl -fsS "$AUTH_URL/health" | grep -q 'auth-service'; then
  ok "auth-service respondeu"
else
  fail "auth-service fora do ar — rode 'make up' primeiro"
fi

# ---- 2. health video --------------------------------------------------------
step "2. Health do video-api"
if curl -fsS "$VIDEO_URL/health" | grep -q 'video-api'; then
  ok "video-api respondeu"
else
  fail "video-api fora do ar — rode 'make up' primeiro"
fi

# ---- 3. registro ------------------------------------------------------------
step "3. Registrar usuário ($EMAIL)"
reg="$(curl -fsS -X POST "$AUTH_URL/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
user_id="$(get_field "$reg" id)"
[ -n "$user_id" ] && ok "usuário criado (id=$user_id)" || fail "falha no registro"

# registro duplicado → 409
code="$(http_code -X POST "$AUTH_URL/register" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
[ "$code" = "409" ] && ok "registro duplicado → 409" || fail "esperado 409, veio $code"

# ---- 4. login ---------------------------------------------------------------
step "4. Login"
login="$(curl -fsS -X POST "$AUTH_URL/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
TOKEN="$(get_field "$login" token)"
[ -n "$TOKEN" ] && ok "token JWT obtido" || fail "login não retornou token"

# senha errada → 401
code="$(http_code -X POST "$AUTH_URL/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"errada123\"}")"
[ "$code" = "401" ] && ok "senha errada → 401" || fail "esperado 401, veio $code"

# ---- 5. gerar vídeo ---------------------------------------------------------
step "5. Gerar vídeo de teste (2s)"
if ffmpeg -y -f lavfi -i testsrc=duration=2:size=320x240:rate=10 \
  -pix_fmt yuv420p "$VIDEO" >/dev/null 2>&1; then
  ok "vídeo gerado ($(stat -f%z "$VIDEO" 2>/dev/null || stat -c%s "$VIDEO") bytes)"
else
  fail "ffmpeg falhou ao gerar o vídeo"
fi

# ---- 6. upload --------------------------------------------------------------
step "6. Upload do vídeo"
up="$(curl -fsS -X POST "$VIDEO_URL/videos" \
  -H "Authorization: Bearer $TOKEN" \
  -F "video=@$VIDEO")"
VIDEO_ID="$(get_field "$up" id)"
[ -n "$VIDEO_ID" ] && ok "upload aceito (202, id=$VIDEO_ID)" || fail "upload falhou"

# ---- 7. aguardar processamento ---------------------------------------------
step "7. Aguardar processamento (ffmpeg + zip)"
status=""
for _ in $(seq 1 "$TIMEOUT_STATUS"); do
  st="$(curl -fsS "$VIDEO_URL/videos/$VIDEO_ID" -H "Authorization: Bearer $TOKEN")"
  status="$(get_field "$st" status)"
  if [ "$status" = "COMPLETED" ]; then break; fi
  if [ "$status" = "FAILED" ]; then fail "processamento terminou em FAILED"; fi
  printf '   ... status: %s\n' "$status"
  sleep 1
done
[ "$status" = "COMPLETED" ] && ok "status COMPLETED" || fail "timeout aguardando COMPLETED"

# ---- 8. listar --------------------------------------------------------------
step "8. Listar vídeos do usuário"
list="$(curl -fsS "$VIDEO_URL/videos" -H "Authorization: Bearer $TOKEN")"
total="$(get_field "$list" total)"
[ "$total" -ge 1 ] && ok "listagem retornou $total vídeo(s)" || fail "listagem vazia"

# ---- 9. download + validar zip ---------------------------------------------
step "9. Download do zip de frames"
curl -fsS "$VIDEO_URL/videos/$VIDEO_ID/download" \
  -H "Authorization: Bearer $TOKEN" -o "$ZIP"
if [ "$(head -c2 "$ZIP")" = "PK" ]; then
  ok "zip válido (signature 'PK')"
else
  fail "download não retornou um arquivo zip"
fi

# ---- 10. cenário de falha ---------------------------------------------------
step "10. Upload de arquivo inválido (falha esperada)"
printf 'isto não é um vídeo\n' > "$BAD_VIDEO"
up_bad="$(curl -fsS -X POST "$VIDEO_URL/videos" \
  -H "Authorization: Bearer $TOKEN" \
  -F "video=@$BAD_VIDEO")"
BAD_ID="$(get_field "$up_bad" id)"
[ -n "$BAD_ID" ] && ok "upload aceito (id=$BAD_ID)" || fail "upload do arquivo inválido falhou"

# ---- 11. aguardar FAILED ----------------------------------------------------
step "11. Aguardar falha no processamento (status FAILED)"
bad_status=""
for _ in $(seq 1 "$TIMEOUT_STATUS"); do
  st="$(curl -fsS "$VIDEO_URL/videos/$BAD_ID" -H "Authorization: Bearer $TOKEN")"
  bad_status="$(get_field "$st" status)"
  if [ "$bad_status" = "FAILED" ]; then break; fi
  printf '   ... status: %s\n' "$bad_status"
  sleep 1
done
[ "$bad_status" = "FAILED" ] && ok "status FAILED" || fail "esperado FAILED, veio $bad_status"

# ---- 12. e-mail de falha (MailHog / Mailpit) -------------------------------
step "12. Verificar e-mail de falha (MailHog/Mailpit)"
email_found=0
for _ in $(seq 1 15); do
  for api in /api/v2/messages /api/v1/messages; do
    if curl -fsS "$MAILHOG_URL$api" 2>/dev/null | grep -q "$BAD_ID"; then
      email_found=1
      break 2
    fi
  done
  sleep 1
done
[ "$email_found" = "1" ] && ok "e-mail de falha enviado (contém o id do vídeo)" \
  || fail "não encontrei e-mail de falha no MailHog/Mailpit"

# ---- 13. listar objetos no S3 ---------------------------------------------
step "13. Listar objetos no S3 (docker exec $S3_CONTAINER)"
s3_listing="$(docker exec "$S3_CONTAINER" wget -qO- --header='Accept: application/json' \
  "http://127.0.0.1:8888/buckets/fiapx/$user_id/?pretty=y" 2>/dev/null || true)"

echo "$s3_listing" \
  | grep -oE '"FullPath": "[^"]+"' \
  | sed 's/.*"FullPath": "//; s/"$//; s#^/buckets/fiapx/##' \
  | grep -v '^\.uploads$' || true

if echo "$s3_listing" | grep -q "$VIDEO_ID"; then
  ok "objetos do vídeo ($VIDEO_ID) estão no S3 (bucket fiapx)"
else
  fail "não encontrei os objetos do vídeo no S3"
fi

# ---- resumo -----------------------------------------------------------------
printf '\n\033[1;32m✅ SMOKE TEST PASSOU — fluxo feliz e de falha OK\033[0m\n'
printf '   vídeo OK:    %s\n' "$VIDEO_ID"
printf '   vídeo FAILED:%s\n' "$BAD_ID"
