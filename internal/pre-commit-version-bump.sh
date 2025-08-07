#!/usr/bin/env bash
set -e

# === КонфИГУРАЦИЯ ===
FILE="version.go"
PATTERN='const Version = "'
REGEX='[0-9]+\.[0-9]+\.[0-9]+(\+[0-9]+)?'

pwd
ls -la 

# === Проверки ===
if [[ ! -f "$FILE" ]]; then
  echo "Файл версии не найден: $FILE"
  exit 1
fi

# === Получение текущей версии ===
version_line=$(grep "$PATTERN" "$FILE")
full_version=$(echo "$version_line" | grep -oE "$REGEX")
version=$(echo "$full_version" | cut -d'+' -f1)
if [[ -z "$version" ]]; then
  echo "Не удалось извлечь версию"
  exit 1
fi

IFS='.' read -r major minor patch <<< "$version"

# === Получение сообщения коммита ===
message=$(git log -1 --pretty=%B)

# === Логика изменения версии ===
if echo "$message" | grep -qi "#major"; then
  major=$((major + 1))
  minor=0
  patch=0
  bump_type="MAJOR"
elif echo "$message" | grep -qi "#minor"; then
  minor=$((minor + 1))
  patch=0
  bump_type="MINOR"
elif echo "$message" | grep -qi "#patch"; then
  patch=$((patch + 1))
  bump_type="PATCH"
else
  bump_type="BUILD-only"
fi

# === Генерация timestamp ===
build_ts=$(date +"%Y%m%d%H%M%S")

# === Новая версия с timestamp ===
new_version="$major.$minor.$patch+$build_ts"

# === Обновление файла ===
# === Сборка полной строки
old_line="${PATTERN}${full_version}\""
new_line="${PATTERN}${new_version}\""
echo "$old_line->$new_line"

# === Заменяем одну строку
sed -i "s|$old_line|$new_line|" "$FILE"

# === Git add ===
git add "$FILE"

# === Лог ===
echo "Версия обновлена ($bump_type): $version → $new_version"
