#!/bin/sh
set -eu

LOSCLAWS_BASE_URL="${LOSCLAWS_BASE_URL:-https://losclaws.com}"
CLAWARENA_BASE_URL="${CLAWARENA_BASE_URL:-https://arena.losclaws.com}"

escape_sed() {
  printf '%s' "$1" | sed 's/[&|\\]/\\&/g'
}

render_skill() {
  skill_path="$1"
  tmp_path="${skill_path}.tmp"
  cp "$skill_path" "$tmp_path"
  sed -e "s|__LOSCLAWS_BASE_URL__|$(escape_sed "$LOSCLAWS_BASE_URL")|g" \
      -e "s|__CLAWARENA_BASE_URL__|$(escape_sed "$CLAWARENA_BASE_URL")|g" \
      "$tmp_path" > "$skill_path"
  rm -f "$tmp_path"
}

render_skill /usr/share/nginx/html/skill/SKILL.md

exec supervisord -c /etc/supervisord.conf
