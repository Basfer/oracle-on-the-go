# gocl

## Usage: gocl [options]

### Options:
  -input, -i string
        SQL file to execute
  -code, -c string
        SQL query to execute
  -output, -o string
        Output file name
  -format, -f string
        Output format (tsv, csv, jira, html, xls, xlsx)
        If not specified, format is determined by output file extension
  -noheader, -H
        Don't output headers
  -help, -h
        Show this help message

### Input sources (in order of priority):
  1. -input (-i) file
  2. -code (-c) query
  3. stdin (pipe or redirect)

### Output formats:
  tsv   - Tab Separated Values (default)
  csv   - Comma Separated Values
  jira  - Jira/Confluence table format
  html  - HTML table format
  xls   - Excel 97-2003 format
  xlsx  - Excel 2007+ format

### Format auto-detection by file extension:
  .tsv, .txt  → tsv
  .csv        → csv
  .html, .htm → html
  .xls        → xls
  .xlsx       → xlsx
  .jira       → jira

### Multiple queries separator: '/' (like in sqlplus)

### Environment variable:
  ORACLE_CONNECTION_STRING - Oracle connection string
    Format: oracle://username:password@hostname:port/service_name
    Example: oracle://scott:tiger@localhost:1521/XE
    With special characters: oracle://user:p%40ssw0rd@host:1521/ORCL

### Examples:
  export ORACLE_CONNECTION_STRING="oracle://user:pass@localhost:1521/XE"
  gocl -c "SELECT * FROM dual" -o result.csv
  gocl -i queries.sql -f html
  echo "SELECT * FROM dual;" | gocl -o results.xlsx
  cat queries.sql | gocl -f jira > output.jira
