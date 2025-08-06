# gocl - Oracle Database CLI Client

[![Build and Release](https://github.com/Basfer/oracle-on-the-go/actions/workflows/go.yml/badge.svg)](https://github.com/Basfer/oracle-on-the-go/actions)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Basfer/oracle-on-the-go)](https://github.com/Basfer/oracle-on-the-go/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/Basfer/oracle-on-the-go/blob/main/LICENSE)

Command-line utility for executing SQL queries against Oracle database with output in various formats.

## Features

- Execute SQL queries against Oracle database
- Support for multiple output formats: TSV, CSV, HTML, Jira Wiki, Excel (XLS, XLSX)
- Ability to execute multiple queries separated by '/' character
- Automatic format detection by output file extension
- Support for reading SQL from files, command line, or stdin
- Creating separate sheets in Excel for each query

## Installation

### From GitHub Releases

Download the appropriate binary for your operating system from the [latest release](https://github.com/Basfer/oracle-on-the-go/releases/latest):

- Linux: `gocl-linux-amd64.tar.gz`
- Windows: `gocl-windows-amd64.zip`
- macOS: `gocl-darwin-amd64.tar.gz`
- Solaris: `gocl-solaris-amd64.tar.gz`
- FreeBSD: `gocl-freebsd-amd64.tar.gz`

### Using the latest build

The latest automatic build is available in the [latest release](https://github.com/Basfer/oracle-on-the-go/releases/tag/latest).

### From source code

Requires Go 1.23 or higher:

```bash
git clone https://github.com/Basfer/oracle-on-the-go.git
cd oracle-on-the-go
go build -o gocl
```

## Quick Start

1. Set the environment variable with Oracle connection string:
   ```bash
   export ORACLE_CONNECTION_STRING="oracle://username:password@hostname:port/service_name"
   ```

2. Execute a simple query:
   ```bash
   gocl -c "SELECT * FROM dual"
   ```

## Usage

### Basic parameters

```
gocl [options]
```

**Parameters:**
- `-input, -i string` - SQL file to execute
- `-code, -c string` - SQL query to execute
- `-output, -o string` - Output file name
- `-format, -f string` - Output format (tsv, csv, jira, html, xls, xlsx)
- `-noheader, -H` - Don't output headers
- `-help, -h` - Show help

### Input sources (in order of priority)

1. SQL file (`-input` or `-i`)
2. SQL query from command line (`-code` or `-c`)
3. Input via stdin (pipe or redirect)

### Output formats

- **tsv** - Tab-separated values (default)
- **csv** - Comma-separated values
- **jira** - Jira/Confluence table format
- **html** - HTML table
- **xls** - Excel 97-2003 format
- **xlsx** - Excel 2007+ format

### Automatic format detection

If format is not specified explicitly, it's determined by the output file extension:
- `.tsv`, `.txt` → tsv
- `.csv` → csv
- `.html`, `.htm` → html
- `.xls` → xls
- `.xlsx` → xlsx
- `.jira` → jira

### Query separation

Multiple SQL queries are separated by '/' character (like in sqlplus):
```sql
SELECT * FROM table1;
/
SELECT count(*) FROM table2; 
/
SELECT sysdate FROM dual;
```

## Examples

### Execute query from command line
```bash
gocl -c "SELECT table_name FROM user_tables WHERE rownum <= 5" -o tables.csv
```

### Execute SQL file with HTML output
```bash
gocl -i queries.sql -f html
```

### Working with pipes and Jira format
```bash
echo "SELECT * FROM dual;" | gocl -o output.jira
```

### Create Excel file with multiple sheets
```bash
gocl -i multiple_queries.sql -o results.xlsx
```

### Read from stdin without specifying source
```bash
cat queries.sql | gocl -o report.html
```

## Oracle Connection String

Connection string format is set through the `ORACLE_CONNECTION_STRING` environment variable:

```
oracle://username:password@hostname:port/service_name
```

**Examples:**
```bash
# Simple connection
export ORACLE_CONNECTION_STRING="oracle://scott:tiger@localhost:1521/XE"

# With special characters escaped in password
export ORACLE_CONNECTION_STRING="oracle://user:p%40ssw0rd@host:1521/ORCL"

# For container database
export ORACLE_CONNECTION_STRING="oracle://user:pass@localhost:1521/ORCLCDB"
```

**Escaping special characters in password:**
- `@` → `%40`
- `:` → `%3A`
- `/` → `%2F`
- `?` → `%3F`
- `#` → `%23`
- `[` → `%5B`
- `]` → `%5D`

## Building

The project uses GitHub Actions for automatic building across various platforms:

- Linux (amd64, arm64)
- Windows (amd64)
- macOS (amd64, arm64)
- Solaris (amd64)
- FreeBSD (amd64)

## License

[MIT License](LICENSE)

## Author

Sergei Badamshin aka BaSF