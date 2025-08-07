package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/sijms/go-ora/v2"
	"github.com/xuri/excelize/v2"
)

type OutputFormat string

const (
	TSV  OutputFormat = "tsv"
	CSV  OutputFormat = "csv"
	HTML OutputFormat = "html"
	JIRA OutputFormat = "jira"
	XLS  OutputFormat = "xls"
	XLSX OutputFormat = "xlsx"
)

type ConnectionParams struct {
	User     string
	Password string
	Server   string
	Port     string
	Service  string
	ConnStr  string
}

type OutputConfig struct {
	Filename string
	Format   OutputFormat
	NoHeader bool
}

type AppParams struct {
	Help        bool
	InputFile   string
	QueryCode   string
	Outputs     []OutputConfig
	ConnectStr  string
	ConnParams  ConnectionParams
	Params      map[string]string
	Interactive bool
}

var outputsList []string
var formatsList []string
var noheaderList []bool

func main() {
	params := parseFlags()

	if params.Help {
		printHelp()
		return
	}

	if err := run(params); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *AppParams {
	var params AppParams
	params.Params = make(map[string]string)

	flag.BoolVar(&params.Help, "help", false, "Show help message")
	flag.BoolVar(&params.Help, "h", false, "Show help message (shorthand)")

	flag.StringVar(&params.InputFile, "input", "", "Input SQL file (default stdin)")
	flag.StringVar(&params.InputFile, "i", "", "Input SQL file (shorthand)")

	flag.StringVar(&params.QueryCode, "code", "", "SQL query to execute")
	flag.StringVar(&params.QueryCode, "c", "", "SQL query to execute (shorthand)")

	flag.StringVar(&params.ConnectStr, "connect", "", "Oracle connection string")
	flag.StringVar(&params.ConnectStr, "C", "", "Oracle connection string (shorthand)")

	flag.StringVar(&params.ConnParams.User, "user", "", "Database username")
	flag.StringVar(&params.ConnParams.User, "u", "", "Database username (shorthand)")

	flag.StringVar(&params.ConnParams.Password, "password", "", "Database password")
	flag.StringVar(&params.ConnParams.Password, "p", "", "Database password (shorthand)")

	flag.StringVar(&params.ConnParams.Server, "server", "", "Database server")
	flag.StringVar(&params.ConnParams.Server, "s", "", "Database server (shorthand)")

	flag.StringVar(&params.ConnParams.Service, "database", "", "Database service name")
	flag.StringVar(&params.ConnParams.Service, "d", "", "Database service name (shorthand)")

	// For multiple outputs
	flag.Var((*stringSlice)(&outputsList), "output", "Output file (can be specified multiple times)")
	flag.Var((*stringSlice)(&outputsList), "o", "Output file (shorthand)")

	// For multiple formats
	flag.Var((*stringSlice)(&formatsList), "format", "Output format for preceding output")
	flag.Var((*stringSlice)(&formatsList), "f", "Output format (shorthand)")

	// For noheader flags
	flag.Var((*boolSlice)(&noheaderList), "noheader", "Don't print column headers")
	flag.Var((*boolSlice)(&noheaderList), "H", "Don't print column headers (shorthand)")

	// Custom parsing for parameters
	flag.Parse()

	// Parse remaining arguments for parameters
	args := flag.Args()
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			params.Params[parts[0]] = parts[1]
		} else {
			fmt.Fprintf(os.Stderr, "Error: Unknown parameter: %s\n", arg)
			printHelp()
			os.Exit(1)
		}
	}

	// Create output configs
	params.Outputs = createOutputConfigs()

	// Check if running interactively
	stat, _ := os.Stdin.Stat()
	params.Interactive = (stat.Mode() & os.ModeCharDevice) != 0

	return &params
}

// Helper types for multiple flag values
type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type boolSlice []bool

func (b *boolSlice) String() string {
	return fmt.Sprintf("%v", *b)
}

func (b *boolSlice) Set(value string) error {
	*b = append(*b, true)
	return nil
}

func createOutputConfigs() []OutputConfig {
	var configs []OutputConfig

	// If no outputs specified, add default stdout
	if len(outputsList) == 0 {
		return []OutputConfig{{Filename: "", Format: TSV}}
	}

	for i, output := range outputsList {
		config := OutputConfig{
			Filename: output,
		}

		// Set format if specified
		if i < len(formatsList) && formatsList[i] != "" {
			config.Format = OutputFormat(formatsList[i])
		}

		// Set noheader if specified
		if i < len(noheaderList) && noheaderList[i] {
			config.NoHeader = true
		}

		configs = append(configs, config)
	}

	return configs
}

func printHelp() {
	helpText := `gocl - Oracle database client

Usage: gocl [options] [parameters]

Options:
  -help, -h               Show this help message
  -input, -i <file>       Input SQL file (default: stdin)
  -code, -c <query>       SQL query to execute directly
  -output, -o <file>      Output file (can be specified multiple times)
  -format, -f <format>    Output format for preceding -o flag
  -noheader, -H           Don't print column headers
  -connect, -C <connstr>  Oracle connection string
  -user, -u <username>    Database username
  -password, -p <password> Database password
  -server, -s <server>    Database server
  -database, -d <service> Database service name

Parameters:
  param=value             Substitution parameters for SQL

Formats:
  tsv, csv, html, jira, xls, xlsx

Connection String Format:
  oracle://user:password@server:port/service

Examples:
  gocl -i query.sql -o result.csv -f csv
  gocl -c "SELECT * FROM dual" -o output.html -f html
  gocl -i query.sql -p param1=value1 param2=value2
`
	fmt.Print(helpText)
}

func run(params *AppParams) error {
	// Build connection string
	connStr, err := buildConnectionString(params)
	if err != nil {
		return fmt.Errorf("connection error: %w", err)
	}

	// Open database connection
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Get input reader
	var reader io.Reader
	if params.QueryCode != "" {
		reader = strings.NewReader(params.QueryCode)
	} else if params.InputFile != "" {
		file, err := os.Open(params.InputFile)
		if err != nil {
			return fmt.Errorf("failed to open input file: %w", err)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	// Process commands
	if err := processCommands(db, reader, params); err != nil {
		return err
	}

	return nil
}

func buildConnectionString(params *AppParams) (string, error) {
	// If connection string is provided directly, use it
	if params.ConnectStr != "" {
		return params.ConnectStr, nil
	}

	// If individual parameters are provided, build connection string
	if params.ConnParams.User != "" && params.ConnParams.Password != "" &&
		params.ConnParams.Server != "" && params.ConnParams.Service != "" {
		return fmt.Sprintf("oracle://%s:%s@%s/%s",
			params.ConnParams.User,
			params.ConnParams.Password,
			params.ConnParams.Server,
			params.ConnParams.Service), nil
	}

	// Try environment variable
	if envConnStr := os.Getenv("ORACLE_CONNECTION_STRING"); envConnStr != "" {
		return envConnStr, nil
	}

	return "", fmt.Errorf("no valid connection parameters provided")
}

func processCommands(db *sql.DB, reader io.Reader, params *AppParams) error {
	scanner := bufio.NewScanner(reader)
	var buffer strings.Builder
	lineNum := 0
	queryIndex := 1

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Print prompt ONLY in interactive mode
		if params.Interactive && buffer.Len() == 0 {
			fmt.Print("SQL> ")
		}

		// Check for command separator
		if isCommandSeparator(line) {
			if buffer.Len() > 0 {
				query := buffer.String()
				if err := executeQuery(db, query, params, queryIndex); err != nil {
					return fmt.Errorf("error executing query at line %d: %w", lineNum, err)
				}
				buffer.Reset()
				queryIndex++
			}
			continue
		}

		// Add line to buffer
		if buffer.Len() > 0 {
			buffer.WriteString("\n")
		}
		buffer.WriteString(line)
	}

	// Process remaining content
	if buffer.Len() > 0 {
		query := buffer.String()
		if err := executeQuery(db, query, params, queryIndex); err != nil {
			return fmt.Errorf("error executing query: %w", err)
		}
	}

	return scanner.Err()
}

func isCommandSeparator(line string) bool {
	// Trim whitespace
	trimmed := strings.TrimSpace(line)

	// Check if line consists only of '/'
	if trimmed == "" {
		return false
	}

	for _, r := range trimmed {
		if r != '/' {
			return false
		}
	}
	return true
}

func executeQuery(db *sql.DB, query string, params *AppParams, queryIndex int) error {
	// Substitute parameters
	finalQuery := substituteParams(query, params.Params)

	// Execute query
	rows, err := db.Query(finalQuery)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Read all rows
	var data [][]string
	for rows.Next() {
		// Create a slice to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to strings
		row := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		data = append(data, row)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Output results
	for _, output := range params.Outputs {
		if err := writeOutput(columns, data, &output, queryIndex); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}

func substituteParams(query string, params map[string]string) string {
	result := query
	for key, value := range params {
		placeholder := "&" + key
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func writeOutput(columns []string, data [][]string, config *OutputConfig, queryIndex int) error {
	// Determine format if not specified
	format := config.Format
	if format == "" {
		if config.Filename != "" {
			format = getFormatFromExtension(config.Filename)
		}
		if format == "" {
			format = TSV
		}
	}

	// Write based on format
	switch format {
	case TSV:
		return writeTSV(config.Filename, columns, data, !config.NoHeader, queryIndex)
	case CSV:
		return writeCSV(config.Filename, columns, data, !config.NoHeader, queryIndex)
	case HTML:
		return writeHTML(config.Filename, columns, data, !config.NoHeader, queryIndex)
	case JIRA:
		return writeJIRA(config.Filename, columns, data, !config.NoHeader, queryIndex)
	case XLS, XLSX:
		return writeExcel(config.Filename, columns, data, !config.NoHeader, queryIndex)
	default:
		return writeTSV(config.Filename, columns, data, !config.NoHeader, queryIndex)
	}
}

func getFormatFromExtension(filename string) OutputFormat {
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".csv"):
		return CSV
	case strings.HasSuffix(strings.ToLower(filename), ".html") ||
		strings.HasSuffix(strings.ToLower(filename), ".htm"):
		return HTML
	case strings.HasSuffix(strings.ToLower(filename), ".xls"):
		return XLS
	case strings.HasSuffix(strings.ToLower(filename), ".xlsx"):
		return XLSX
	default:
		return TSV
	}
}

func writeTSV(filename string, columns []string, data [][]string, withHeader bool, queryIndex int) error {
	var file *os.File
	var err error

	// For subsequent queries, append to file
	if queryIndex > 1 && filename != "" {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	} else if filename == "" {
		file = os.Stdout
	} else {
		file, err = os.Create(filename)
	}

	if err != nil {
		return err
	}

	if filename != "" {
		defer file.Close()
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Add separator between results
	if queryIndex > 1 {
		fmt.Fprintln(writer, "")
	}

	if withHeader {
		fmt.Fprintln(writer, strings.Join(columns, "\t"))
	}

	for _, row := range data {
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	return nil
}

func writeCSV(filename string, columns []string, data [][]string, withHeader bool, queryIndex int) error {
	var file *os.File
	var err error

	// For subsequent queries, append to file
	if queryIndex > 1 && filename != "" {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	} else if filename == "" {
		file = os.Stdout
	} else {
		file, err = os.Create(filename)
	}

	if err != nil {
		return err
	}

	if filename != "" {
		defer file.Close()
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Add separator between results
	if queryIndex > 1 {
		fmt.Fprintln(writer, "")
	}

	if withHeader {
		fmt.Fprintln(writer, strings.Join(columns, ","))
	}

	for _, row := range data {
		fmt.Fprintln(writer, strings.Join(row, ","))
	}

	return nil
}

func writeHTML(filename string, columns []string, data [][]string, withHeader bool, queryIndex int) error {
	// For first query, create new file with header
	if queryIndex == 1 {
		return writeHTMLNew(filename, columns, data, withHeader)
	}

	// For subsequent queries, append to existing file
	return writeHTMLAppend(filename, columns, data, withHeader)
}

func writeHTMLNew(filename string, columns []string, data [][]string, withHeader bool) error {
	var file *os.File
	var err error

	if filename == "" {
		file = os.Stdout
	} else {
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write HTML header
	header := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Query Results</title>
    <style>
        table { border-collapse: collapse; margin-bottom: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
`
	fmt.Fprint(writer, header)

	// Write first table
	if err := writeHTMLTable(writer, columns, data, withHeader); err != nil {
		return err
	}

	// Write HTML footer (will be updated when appending)
	footer := `
</body>
</html>
`
	fmt.Fprint(writer, footer)

	return nil
}

func writeHTMLAppend(filename string, columns []string, data [][]string, withHeader bool) error {
	if filename == "" {
		// For stdout, just write the table
		return writeHTMLTable(os.Stdout, columns, data, withHeader)
	}

	// Read existing file
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Find position to insert new table (before </body>)
	contentStr := string(content)
	pos := strings.LastIndex(contentStr, "</body>")
	if pos == -1 {
		return fmt.Errorf("invalid HTML file format")
	}

	// Create new content
	newContent := contentStr[:pos]

	// Add the new table
	var tableBuf strings.Builder
	writer := bufio.NewWriter(&tableBuf)
	if err := writeHTMLTable(writer, columns, data, withHeader); err != nil {
		return err
	}
	writer.Flush()

	newContent += tableBuf.String()
	newContent += contentStr[pos:]

	// Write back to file
	return os.WriteFile(filename, []byte(newContent), 0644)
}

func writeHTMLTable(writer io.Writer, columns []string, data [][]string, withHeader bool) error {
	fmt.Fprintln(writer, "    <table>")

	if withHeader {
		fmt.Fprintln(writer, "        <thead>")
		fmt.Fprintln(writer, "            <tr>")
		for _, col := range columns {
			fmt.Fprintf(writer, "                <th>%s</th>\n", col)
		}
		fmt.Fprintln(writer, "            </tr>")
		fmt.Fprintln(writer, "        </thead>")
	}

	fmt.Fprintln(writer, "        <tbody>")
	for _, row := range data {
		fmt.Fprintln(writer, "            <tr>")
		for _, cell := range row {
			fmt.Fprintf(writer, "                <td>%s</td>\n", cell)
		}
		fmt.Fprintln(writer, "            </tr>")
	}
	fmt.Fprintln(writer, "        </tbody>")
	fmt.Fprintln(writer, "    </table>")

	return nil
}

func writeJIRA(filename string, columns []string, data [][]string, withHeader bool, queryIndex int) error {
	var file *os.File
	var err error

	// For subsequent queries, append to file
	if queryIndex > 1 && filename != "" {
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	} else if filename == "" {
		file = os.Stdout
	} else {
		file, err = os.Create(filename)
	}

	if err != nil {
		return err
	}

	if filename != "" {
		defer file.Close()
	}

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Add separator between results
	if queryIndex > 1 {
		fmt.Fprintln(writer, "")
	}

	if withHeader {
		// Header row
		fmt.Fprint(writer, "||")
		for _, col := range columns {
			fmt.Fprintf(writer, " %s |", col)
		}
		fmt.Fprintln(writer)
	}

	// Data rows
	for _, row := range data {
		fmt.Fprint(writer, "|")
		for _, cell := range row {
			fmt.Fprintf(writer, " %s |", cell)
		}
		fmt.Fprintln(writer)
	}

	return nil
}

func writeExcel(filename string, columns []string, data [][]string, withHeader bool, queryIndex int) error {
	var f *excelize.File
	var err error

	// For subsequent queries, open existing file
	if queryIndex > 1 {
		f, err = excelize.OpenFile(filename)
		if err != nil {
			return err
		}
	} else {
		f = excelize.NewFile()
		// Remove default sheet
		f.DeleteSheet("Sheet1")
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing Excel file: %v\n", err)
		}
	}()

	// Create new sheet for this query
	sheetName := fmt.Sprintf("Results%d", queryIndex)
	f.NewSheet(sheetName)

	// Write header if needed
	if withHeader {
		for i, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, col)
		}
	}

	// Write data
	for i, row := range data {
		for j, cell := range row {
			rowNum := i + 1
			if withHeader {
				rowNum++
			}
			cellName, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			f.SetCellValue(sheetName, cellName, cell)
		}
	}

	// Save file
	if err := f.SaveAs(filename); err != nil {
		return fmt.Errorf("failed to save Excel file: %w", err)
	}

	return nil
}
