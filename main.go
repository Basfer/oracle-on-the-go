package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

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

	// We'll handle multiple outputs manually
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

	// Custom parsing for multiple outputs and parameters
	flag.Parse()

	// Parse remaining arguments for outputs and parameters
	args := flag.Args()
	var outputs []OutputConfig
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-output", "-o":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: -output flag requires a filename\n")
				printHelp()
				os.Exit(1)
			}
			outputs = append(outputs, OutputConfig{Filename: args[i+1]})
			i += 2
		case "-format", "-f":
			if len(outputs) == 0 {
				fmt.Fprintf(os.Stderr, "Error: -format must follow -output flag\n")
				printHelp()
				os.Exit(1)
			}
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "Error: -format flag requires a format\n")
				printHelp()
				os.Exit(1)
			}
			// Update the last output config
			idx := len(outputs) - 1
			outputs[idx].Format = OutputFormat(args[i+1])
			i += 2
		case "-noheader", "-H":
			if len(outputs) == 0 {
				fmt.Fprintf(os.Stderr, "Error: -noheader must follow -output flag\n")
				printHelp()
				os.Exit(1)
			}
			// Update the last output config
			idx := len(outputs) - 1
			outputs[idx].NoHeader = true
			i++
		default:
			// Handle parameters like param1=value
			if strings.Contains(args[i], "=") {
				parts := strings.SplitN(args[i], "=", 2)
				params.Params[parts[0]] = parts[1]
			} else {
				fmt.Fprintf(os.Stderr, "Error: Unknown parameter: %s\n", args[i])
				printHelp()
				os.Exit(1)
			}
			i++
		}
	}

	// If no outputs specified, add default stdout
	if len(outputs) == 0 {
		outputs = append(outputs, OutputConfig{Filename: "", Format: TSV}) // stdout
	}
	params.Outputs = outputs

	// Check if running interactively
	stat, _ := os.Stdin.Stat()
	params.Interactive = (stat.Mode() & os.ModeCharDevice) != 0

	return &params
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

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Print prompt in interactive mode
		if params.Interactive && buffer.Len() == 0 {
			fmt.Print("SQL> ")
		}

		// Check for command separator
		if isCommandSeparator(line) {
			if buffer.Len() > 0 {
				query := buffer.String()
				if err := executeQuery(db, query, params); err != nil {
					return fmt.Errorf("error executing query at line %d: %w", lineNum, err)
				}
				buffer.Reset()
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
		if err := executeQuery(db, query, params); err != nil {
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

func executeQuery(db *sql.DB, query string, params *AppParams) error {
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
		if err := writeOutput(columns, data, &output); err != nil {
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

func writeOutput(columns []string, data [][]string, config *OutputConfig) error {
	var writer io.Writer

	// Determine output writer
	if config.Filename == "" {
		writer = os.Stdout
	} else {
		file, err := os.Create(config.Filename)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		writer = file
	}

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
		return writeTSV(writer, columns, data, !config.NoHeader)
	case CSV:
		return writeCSV(writer, columns, data, !config.NoHeader)
	case HTML:
		return writeHTML(writer, columns, data, !config.NoHeader)
	case JIRA:
		return writeJIRA(writer, columns, data, !config.NoHeader)
	case XLS, XLSX:
		return writeExcel(config.Filename, columns, data, !config.NoHeader)
	default:
		return writeTSV(writer, columns, data, !config.NoHeader)
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

func writeTSV(writer io.Writer, columns []string, data [][]string, withHeader bool) error {
	if withHeader {
		fmt.Fprintln(writer, strings.Join(columns, "\t"))
	}

	for _, row := range data {
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	return nil
}

func writeCSV(writer io.Writer, columns []string, data [][]string, withHeader bool) error {
	if withHeader {
		fmt.Fprintln(writer, strings.Join(columns, ","))
	}

	for _, row := range data {
		fmt.Fprintln(writer, strings.Join(row, ","))
	}

	return nil
}

func writeHTML(writer io.Writer, columns []string, data [][]string, withHeader bool) error {
	htmlTemplate := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Query Results</title>
    <style>
        table { border-collapse: collapse; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <table>
        {{if .WithHeader}}
        <thead>
            <tr>
                {{range .Columns}}
                <th>{{.}}</th>
                {{end}}
            </tr>
        </thead>
        {{end}}
        <tbody>
            {{range .Data}}
            <tr>
                {{range .}}
                <td>{{.}}</td>
                {{end}}
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
`

	tmpl, err := template.New("html").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(writer, map[string]interface{}{
		"Columns":    columns,
		"Data":       data,
		"WithHeader": withHeader,
	})
}

func writeJIRA(writer io.Writer, columns []string, data [][]string, withHeader bool) error {
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

func writeExcel(filename string, columns []string, data [][]string, withHeader bool) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing Excel file: %v\n", err)
		}
	}()

	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

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
