# gocl - Oracle Database CLI Client

[![Build and Release](https://github.com/Basfer/gocl/actions/workflows/go.yml/badge.svg)](https://github.com/Basfer/gocl/actions)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Basfer/gocl)](https://github.com/Basfer/gocl/releases)
[![License](https://img.shields.io/github/license/Basfer/gocl)](https://github.com/Basfer/gocl/blob/main/LICENSE)

Командная утилита для выполнения SQL-запросов к базе данных Oracle с выводом результатов в различных форматах.

## Возможности

- Выполнение SQL-запросов к Oracle базе данных
- Поддержка нескольких форматов вывода: TSV, CSV, HTML, Jira Wiki, Excel (XLS, XLSX)
- Возможность выполнения нескольких запросов с разделением символом '/'
- Автоматическое определение формата по расширению выходного файла
- Поддержка чтения SQL из файлов, командной строки или stdin
- Создание отдельных листов в Excel для каждого запроса

## Установка

### Из релизов GitHub

Скачайте подходящий бинарный файл для вашей операционной системы из [последнего релиза](https://github.com/yourusername/gocl/releases/latest):

- Linux: `gocl-linux-amd64.tar.gz`
- Windows: `gocl-windows-amd64.zip`
- macOS: `gocl-darwin-amd64.tar.gz`
- Solaris: `gocl-solaris-amd64.tar.gz`
- FreeBSD: `gocl-freebsd-amd64.tar.gz`

### Использование latest-сборки

Последняя автоматическая сборка доступна в [релизе latest](https://github.com/yourusername/gocl/releases/tag/latest).

### Из исходного кода

Требуется Go 1.19 или выше:

```bash
git clone https://github.com/yourusername/gocl.git
cd gocl
go build -o gocl
```

## Быстрый старт

1. Установите переменную окружения с строкой подключения к Oracle:
   ```bash
   export ORACLE_CONNECTION_STRING="oracle://username:password@hostname:port/service_name"
   ```

2. Выполните простой запрос:
   ```bash
   gocl -c "SELECT * FROM dual"
   ```

## Использование

### Основные параметры

```
gocl [options]
```

**Параметры:**
- `-input, -i string` - SQL файл для выполнения
- `-code, -c string` - SQL запрос для выполнения
- `-output, -o string` - Имя выходного файла
- `-format, -f string` - Формат вывода (tsv, csv, jira, html, xls, xlsx)
- `-noheader, -H` - Не выводить заголовки
- `-help, -h` - Показать справку

### Источники ввода (в порядке приоритета)

1. Файл SQL (`-input` или `-i`)
2. SQL запрос из командной строки (`-code` или `-c`)
3. Ввод через stdin (pipe или redirect)

### Форматы вывода

- **tsv** - Значения, разделенные табуляцией (по умолчанию)
- **csv** - Значения, разделенные запятыми
- **jira** - Формат таблиц Jira/Confluence
- **html** - HTML таблица
- **xls** - Формат Excel 97-2003
- **xlsx** - Формат Excel 2007+

### Автоопределение формата

Если формат не указан явно, он определяется по расширению выходного файла:
- `.tsv`, `.txt` → tsv
- `.csv` → csv
- `.html`, `.htm` → html
- `.xls` → xls
- `.xlsx` → xlsx
- `.jira` → jira

### Разделение запросов

Несколько SQL-запросов разделяются символом `/` (как в sqlplus):
```sql
SELECT * FROM table1; /
SELECT count(*) FROM table2; /
SELECT sysdate FROM dual;
```

## Примеры

### Выполнение запроса из командной строки
```bash
gocl -c "SELECT table_name FROM user_tables WHERE rownum <= 5" -o tables.csv
```

### Выполнение SQL файла с выводом в HTML
```bash
gocl -i queries.sql -f html
```

### Работа с пайпами и Jira форматом
```bash
echo "SELECT * FROM dual; / SELECT 1 FROM dual;" | gocl -o output.jira
```

### Создание Excel файла с несколькими листами
```bash
gocl -i multiple_queries.sql -o results.xlsx
```

### Чтение из stdin без указания источника
```bash
cat queries.sql | gocl -o report.html
```

## Строка подключения к Oracle

Формат строки подключения задается через переменную окружения `ORACLE_CONNECTION_STRING`:

```
oracle://username:password@hostname:port/service_name
```

**Примеры:**
```bash
# Простое подключение
export ORACLE_CONNECTION_STRING="oracle://scott:tiger@localhost:1521/XE"

# С экранированием специальных символов в пароле
export ORACLE_CONNECTION_STRING="oracle://user:p%40ssw0rd@host:1521/ORCL"

# Для контейнерной базы данных
export ORACLE_CONNECTION_STRING="oracle://user:pass@localhost:1521/ORCLCDB"
```

**Экранирование специальных символов в пароле:**
- `@` → `%40`
- `:` → `%3A`
- `/` → `%2F`
- `?` → `%3F`
- `#` → `%23`
- `[` → `%5B`
- `]` → `%5D`

## Сборка

Проект использует GitHub Actions для автоматической сборки под различные платформы:

- Linux (amd64, arm64)
- Windows (amd64, 386)
- macOS (amd64, arm64)
- Solaris (amd64)
- FreeBSD (amd64)

## Лицензия

[MIT License](LICENSE)

## Автор

Sergei Badamshin 
```
