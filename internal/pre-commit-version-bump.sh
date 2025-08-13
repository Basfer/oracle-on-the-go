#!/usr/bin/env bash
set -e

# === Конфигурация ===
FILE="version.go"
PATTERN='const Version = "'
VERSION_RE='[0-9]+\.[0-9]+\.[0-9]+'
FULL_RE='[0-9]+\.[0-9]+\.[0-9]+\+[0-9]+'

# === Извлечение строки версии ===
version_line=$(grep "$PATTERN" "$FILE")
current_full=$(echo "$version_line" | grep -oE "$FULL_RE")

# Если версия без +timestamp
if [[ -z "$current_full" ]]; then
  current_base=$(echo "$version_line" | grep -oE "$VERSION_RE")
else
  current_base=$(echo "$current_full" | cut -d'+' -f1)
fi

IFS='.' read -r major minor patch <<< "$current_base"

# === Получение сообщения коммита ===
message=$(git log -1 --pretty=%B)

# === Определение типа обновления ===
if echo "$message" | grep -qi "#major"; then
  major=$((major + 1)); minor=0; patch=0; bump_type="MAJOR"
elif echo "$message" | grep -qi "#minor"; then
  minor=$((minor + 1)); patch=0; bump_type="MINOR"
elif echo "$message" | grep -qi "#patch"; then
  patch=$((patch + 1)); bump_type="PATCH"
else
  bump_type="BUILD-only"
fi

# === timestamp
build_ts=$(date +"%Y%m%d%H%M%S")
new_version="$major.$minor.$patch+$build_ts"

# === Строки для подстановки
old_line=$(grep "$PATTERN" "$FILE")
new_line="${PATTERN}${new_version}\""

# === Заменить строку
sed -i "s|$old_line|$new_line|" "$FILE"
git add "$FILE"

# === Лог
echo "Версия обновлена ($bump_type): $current_base → $new_version"
