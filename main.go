package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/sijms/go-ora/v2"
	"github.com/xuri/excelize/v2"
)

type Config struct {
	inputFile  string
	queryCode  string
	outputFile string
	format     string
	noHeader   bool
	help       bool
	params     map[string]string
}

func main() {
	// ... существующий код инициализации ...

	// Получаем SQL команды
	var queries []string
	var err error

	stat, _ := os.Stdin.Stat()
	if config.inputFile == "" && config.queryCode == "" && len(flag.Args()) == 0 &&
		(stat.Mode()&os.ModeCharDevice) != 0 {
		// Интерактивный режим
		queries, err = getQueriesFromInteractiveMode(db, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting queries: %v\n\n", err)
			showHelp()
			os.Exit(1)
		}
		return // Завершаем программу после интерактивного режима
	} else {
		// Неинтерактивный режим
		queries, err = getQueries(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting queries: %v\n\n", err)
			showHelp()
			os.Exit(1)
		}
	}

	// ... остальной код выполнения запросов ...
}

func getQueriesFromInteractiveMode(db *sql.DB, config Config) error {
	fmt.Println("gocl interactive mode. Type SQL commands, use '/' to execute, 'exit' to quit.")
	var currentQuery strings.Builder

	fmt.Print("SQL> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		// Проверяем команды выхода
		if strings.ToLower(strings.TrimSpace(line)) == "exit" ||
			strings.ToLower(strings.TrimSpace(line)) == "quit" {
			break
		}

		// Удаляем пробельные символы с начала и конца строки
		trimmedLine := strings.TrimSpace(line)

		// Проверяем, является ли строка разделителем команд
		if trimmedLine != "" && strings.Trim(trimmedLine, "/") == "" {
			// Это строка-разделитель (содержит только символы "/")
			// Выполняем текущий запрос
			if currentQuery.Len() > 0 {
				query := strings.TrimSpace(currentQuery.String())
				if query != "" {
					query = strings.TrimSuffix(query, ";")
					query = strings.TrimSpace(query)

					// Применяем подстановку параметров
					query = substituteParams(query, config.params)

					// Выполняем запрос
					if err := executeInteractiveQuery(db, query, config); err != nil {
						fmt.Fprintf(os.Stderr, "Error executing query: %v\n", err)
					}
				}
				currentQuery.Reset()
			}
			fmt.Print("SQL> ")
		} else {
			// Это обычная строка, добавляем её к текущему запросу
			if currentQuery.Len() > 0 {
				currentQuery.WriteString("\n")
			}
			currentQuery.WriteString(line)
		}
	}

	// Выполняем последний запрос, если он есть
	if currentQuery.Len() > 0 {
		query := strings.TrimSpace(currentQuery.String())
		if query != "" {
			query = strings.TrimSuffix(query, ";")
			query = strings.TrimSpace(query)

			// Применяем подстановку параметров
			query = substituteParams(query, config.params)

			// Выполняем запрос
			if err := executeInteractiveQuery(db, query, config); err != nil {
				fmt.Fprintf(os.Stderr, "Error executing query: %v\n", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %v", err)
	}

	return nil
}

func executeInteractiveQuery(db *sql.DB, query string, config Config) error {
	// Пропускаем пустые запросы
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	// Выполняем запрос
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	// Получаем имена колонок
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %v", err)
	}

	// Для интерактивного режима всегда используем TSV формат на stdout
	configCopy := config
	configCopy.format = "tsv"

	// Выводим результаты
	return executeQueryDefault(os.Stdout, configCopy, columns, rows)
}

func substituteParams(query string, params map[string]string) string {
	if len(params) == 0 {
		return query
	}

	result := query
	for paramName, paramValue := range params {
		placeholder := "&" + paramName
		result = strings.ReplaceAll(result, placeholder, paramValue)
	}

	return result
}

func showHelp() {
	fmt.Fprintf(os.Stderr, "Usage: gocl [options] -- [parameters]\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -input, -i string\n")
	fmt.Fprintf(os.Stderr, "        SQL file to execute\n")
	fmt.Fprintf(os.Stderr, "  -code, -c string\n")
	fmt.Fprintf(os.Stderr, "        SQL query to execute\n")
	fmt.Fprintf(os.Stderr, "  -output, -o string\n")
	fmt.Fprintf(os.Stderr, "        Output file name\n")
	fmt.Fprintf(os.Stderr, "  -format, -f string\n")
	fmt.Fprintf(os.Stderr, "        Output format (tsv, csv, jira, html, xls, xlsx)\n")
	fmt.Fprintf(os.Stderr, "        If not specified, format is determined by output file extension\n")
	fmt.Fprintf(os.Stderr, "  -noheader, -H\n")
	fmt.Fprintf(os.Stderr, "        Don't output headers\n")
	fmt.Fprintf(os.Stderr, "  -help, -h\n")
	fmt.Fprintf(os.Stderr, "        Show this help message\n\n")

	fmt.Fprintf(os.Stderr, "Parameters:\n")
	fmt.Fprintf(os.Stderr, "  Parameters are specified after '--' separator\n")
	fmt.Fprintf(os.Stderr, "  Format: -paramName=value or paramName=value\n")
	fmt.Fprintf(os.Stderr, "  Example: gocl -i query.sql -- param1=value1 param2=\"value with spaces\"\n\n")

	fmt.Fprintf(os.Stderr, "Input sources (in order of priority):\n")
	fmt.Fprintf(os.Stderr, "  1. -input (-i) file\n")
	fmt.Fprintf(os.Stderr, "  2. -code (-c) query\n")
	fmt.Fprintf(os.Stderr, "  3. stdin (pipe or redirect)\n\n")

	fmt.Fprintf(os.Stderr, "Output formats:\n")
	fmt.Fprintf(os.Stderr, "  tsv   - Tab Separated Values (default)\n")
	fmt.Fprintf(os.Stderr, "  csv   - Comma Separated Values\n")
	fmt.Fprintf(os.Stderr, "  jira  - Jira/Confluence table format\n")
	fmt.Fprintf(os.Stderr, "  html  - HTML table format\n")
	fmt.Fprintf(os.Stderr, "  xls   - Excel 97-2003 format\n")
	fmt.Fprintf(os.Stderr, "  xlsx  - Excel 2007+ format\n\n")

	fmt.Fprintf(os.Stderr, "Format auto-detection by file extension:\n")
	fmt.Fprintf(os.Stderr, "  .tsv, .txt  → tsv\n")
	fmt.Fprintf(os.Stderr, "  .csv        → csv\n")
	fmt.Fprintf(os.Stderr, "  .html, .htm → html\n")
	fmt.Fprintf(os.Stderr, "  .xls        → xls\n")
	fmt.Fprintf(os.Stderr, "  .xlsx       → xlsx\n")
	fmt.Fprintf(os.Stderr, "  .jira       → jira\n\n")

	fmt.Fprintf(os.Stderr, "Multiple queries separator: '/' (like in sqlplus)\n\n")

	fmt.Fprintf(os.Stderr, "Environment variable:\n")
	fmt.Fprintf(os.Stderr, "  ORACLE_CONNECTION_STRING - Oracle connection string\n")
	fmt.Fprintf(os.Stderr, "    Format: oracle://username:password@hostname:port/service_name\n")
	fmt.Fprintf(os.Stderr, "    Example: oracle://scott:tiger@localhost:1521/XE\n")
	fmt.Fprintf(os.Stderr, "    With special characters: oracle://user:p%%40ssw0rd@host:1521/ORCL\n\n")

	fmt.Fprintf(os.Stderr, "Examples:\n")
	fmt.Fprintf(os.Stderr, "  export ORACLE_CONNECTION_STRING=\"oracle://user:pass@localhost:1521/XE\"\n")
	fmt.Fprintf(os.Stderr, "  gocl -c \"SELECT * FROM dual\" -o result.csv\n")
	fmt.Fprintf(os.Stderr, "  gocl -i queries.sql -f html\n")
	fmt.Fprintf(os.Stderr, "  echo \"SELECT * FROM dual; / SELECT 1 FROM dual;\" | gocl -o results.xlsx\n")
	fmt.Fprintf(os.Stderr, "  cat queries.sql | gocl -f jira > output.jira\n")
	fmt.Fprintf(os.Stderr, "  gocl -i query.sql -- param1=value1 param2=\"value with spaces\" param3=1000\n")
}

func determineFormat(formatFlag string, outputFile string) string {
	// Если формат явно указан через флаг, используем его
	if formatFlag != "" {
		return formatFlag
	}

	// Если формат не указан, пытаемся определить по расширению файла
	if outputFile != "" {
		ext := strings.ToLower(filepath.Ext(outputFile))
		switch ext {
		case ".csv":
			return "csv"
		case ".html", ".htm":
			return "html"
		case ".xls":
			return "xls"
		case ".xlsx":
			return "xlsx"
		case ".jira":
			return "jira"
		default:
			// По умолчанию TSV для любых других расширений или отсутствия расширения
			return "tsv"
		}
	}

	// Если нет ни флага формата, ни выходного файла, используем TSV по умолчанию
	return "tsv"
}

func getQueries(config Config) ([]string, error) {
	// Если указан файл
	if config.inputFile != "" {
		return getQueriesFromFile(config.inputFile)
	} else if config.queryCode != "" {
		// Если указан код напрямую
		return getQueriesFromString(config.queryCode)
	} else {
		// Читаем из stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// Интерактивный режим - построчный ввод
			return getQueriesFromInteractiveInput()
		} else {
			// Неинтерактивный режим - читаем все данные
			return getQueriesFromStdin()
		}
	}
}

func getQueriesFromInteractiveInput() ([]string, error) {
	var queries []string
	var currentQuery strings.Builder

	fmt.Print("SQL> ")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()

		// Удаляем пробельные символы с начала и конца строки
		trimmedLine := strings.TrimSpace(line)

		// Проверяем, является ли строка разделителем команд
		if trimmedLine != "" && strings.Trim(trimmedLine, "/") == "" {
			// Это строка-разделитель (содержит только символы "/")
			// Завершаем текущий запрос
			if currentQuery.Len() > 0 {
				query := strings.TrimSpace(currentQuery.String())
				if query != "" {
					queries = append(queries, query)
					// Выполняем запрос немедленно в интерактивном режиме
					// Но в этой функции просто собираем запросы
				}
				currentQuery.Reset()
			}
			fmt.Print("SQL> ")
		} else {
			// Это обычная строка, добавляем её к текущему запросу
			if currentQuery.Len() > 0 {
				currentQuery.WriteString("\n")
			}
			currentQuery.WriteString(line)
		}
	}

	// Не забываем добавить последний запрос, если он есть
	if currentQuery.Len() > 0 {
		query := strings.TrimSpace(currentQuery.String())
		if query != "" {
			queries = append(queries, query)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from stdin: %v", err)
	}

	// Очищаем команды от точки с запятой в конце
	var result []string
	for _, query := range queries {
		// Удаляем точку с запятой в конце, если она есть
		query = strings.TrimSuffix(query, ";")
		// Еще раз удаляем пробельные символы
		query = strings.TrimSpace(query)

		if query != "" {
			result = append(result, query)
		}
	}

	return result, nil
}

func getQueriesFromFile(filename string) ([]string, error) {
	var lines []string

	// Читаем файл построчно
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}

	return processLines(lines)
}

func getQueriesFromString(content string) ([]string, error) {
	// Разбиваем код на строки
	lines := strings.Split(content, "\n")
	return processLines(lines)
}

func getQueriesFromStdin() ([]string, error) {
	var lines []string

	// Читаем из stdin построчно
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from stdin: %v", err)
	}

	return processLines(lines)
}

func processLines(lines []string) ([]string, error) {
	// Обрабатываем строки и разделяем команды
	var queries []string
	var currentQuery strings.Builder

	for _, line := range lines {
		// Удаляем пробельные символы с начала и конца строки
		trimmedLine := strings.TrimSpace(line)

		// Проверяем, является ли строка разделителем команд
		if trimmedLine != "" && strings.Trim(trimmedLine, "/") == "" {
			// Это строка-разделитель (содержит только символы "/")
			// Завершаем текущий запрос
			if currentQuery.Len() > 0 {
				query := strings.TrimSpace(currentQuery.String())
				if query != "" {
					queries = append(queries, query)
				}
				currentQuery.Reset()
			}
		} else {
			// Это обычная строка, добавляем её к текущему запросу
			if currentQuery.Len() > 0 {
				currentQuery.WriteString("\n")
			}
			currentQuery.WriteString(line)
		}
	}

	// Не забываем добавить последний запрос, если он есть
	if currentQuery.Len() > 0 {
		query := strings.TrimSpace(currentQuery.String())
		if query != "" {
			queries = append(queries, query)
		}
	}

	// Очищаем команды от точки с запятой в конце
	var result []string
	for _, query := range queries {
		// Удаляем точку с запятой в конце, если она есть
		query = strings.TrimSuffix(query, ";")
		// Еще раз удаляем пробельные символы
		query = strings.TrimSpace(query)

		if query != "" {
			result = append(result, query)
		}
	}

	return result, nil
}

func executeTextQueries(db *sql.DB, queries []string, config Config) error {
	// Определяем выходной поток
	var output io.Writer
	if config.outputFile != "" {
		file, err := os.Create(config.outputFile)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}
		defer file.Close()
		output = file
	} else {
		output = os.Stdout
	}

	// Для HTML формата выводим начальный тег, если это не файл
	if config.format == "html" && config.outputFile == "" {
		fmt.Fprintln(output, "<!DOCTYPE html>")
		fmt.Fprintln(output, "<html>")
		fmt.Fprintln(output, "<head>")
		fmt.Fprintln(output, "    <meta charset=\"UTF-8\">")
		fmt.Fprintln(output, "    <title>Query Results</title>")
		fmt.Fprintln(output, "    <style>")
		fmt.Fprintln(output, "        table { border-collapse: collapse; margin: 20px 0; }")
		fmt.Fprintln(output, "        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }")
		fmt.Fprintln(output, "        th { background-color: #f2f2f2; }")
		fmt.Fprintln(output, "    </style>")
		fmt.Fprintln(output, "</head>")
		fmt.Fprintln(output, "<body>")
	}

	// Выполняем запросы
	for i, query := range queries {
		if i > 0 && config.outputFile == "" && config.format != "html" {
			fmt.Fprintln(output, "") // Пустая строка между результатами для не-HTML форматов
		}

		if err := executeTextQuery(db, query, output, config); err != nil {
			fmt.Fprintf(os.Stderr, "Error executing query %v: %v\n", query, err)
			continue
		}
	}

	// Для HTML формата выводим закрывающие теги, если это не файл
	if config.format == "html" && config.outputFile == "" {
		fmt.Fprintln(output, "</body>")
		fmt.Fprintln(output, "</html>")
	}

	return nil
}

func executeTextQuery(db *sql.DB, query string, output io.Writer, config Config) error {
	// Пропускаем пустые запросы
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	// Выполняем запрос
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	// Получаем имена колонок
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %v", err)
	}

	// Выводим данные в зависимости от формата
	switch config.format {
	case "html":
		return executeQueryHTML(db, query, output, config, columns, rows)
	default:
		return executeQueryDefault(output, config, columns, rows)
	}
}

func executeExcelQueries(db *sql.DB, queries []string, config Config) error {
	// Создаем новый Excel файл
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing Excel file: %v\n", err)
		}
	}()

	// Удаляем дефолтный лист
	f.DeleteSheet(f.GetSheetName(0))

	// Выполняем каждый запрос и сохраняем в отдельную вкладку
	for i, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}

		// Создаем имя вкладки
		sheetName := fmt.Sprintf("Query%d", i+1)
		if i == 0 {
			f.SetSheetName(f.GetSheetName(0), sheetName)
		} else {
			f.NewSheet(sheetName)
		}

		// Выполняем запрос
		rows, err := db.Query(query)
		if err != nil {
			return fmt.Errorf("error executing query %d: %v", i+1, err)
		}

		// Получаем имена колонок
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return fmt.Errorf("error getting columns for query %d: %v", i+1, err)
		}

		// Записываем заголовки, если нужно
		rowNum := 1
		if !config.noHeader && len(columns) > 0 {
			for colIdx, colName := range columns {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				f.SetCellValue(sheetName, cell, colName)
			}
			rowNum++
		}

		// Обрабатываем строки результата
		for rows.Next() {
			// Создаем слайс для значений
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for j := range values {
				valuePtrs[j] = &values[j]
			}

			// Сканируем значения
			if err := rows.Scan(valuePtrs...); err != nil {
				rows.Close()
				return fmt.Errorf("error scanning row for query %d: %v", i+1, err)
			}

			// Записываем значения в ячейки
			for colIdx, v := range values {
				cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowNum)
				if v == nil {
					f.SetCellValue(sheetName, cell, "")
				} else {
					f.SetCellValue(sheetName, cell, v)
				}
			}
			rowNum++
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("error iterating rows for query %d: %v", i+1, err)
		}

		rows.Close()
	}

	// Сохраняем файл
	if err := f.SaveAs(config.outputFile); err != nil {
		return fmt.Errorf("error saving Excel file: %v", err)
	}

	return nil
}

func executeQueryHTML(db *sql.DB, query string, output io.Writer, config Config, columns []string, rows *sql.Rows) error {
	// Начало таблицы HTML
	fmt.Fprintln(output, "<table>")

	// Выводим заголовки, если нужно
	if !config.noHeader && len(columns) > 0 {
		fmt.Fprintln(output, "    <thead>")
		fmt.Fprint(output, "        <tr>")
		for _, col := range columns {
			fmt.Fprintf(output, "<th>%s</th>", escapeHTML(col))
		}
		fmt.Fprintln(output, "</tr>")
		fmt.Fprintln(output, "    </thead>")
	}

	// Начало тела таблицы
	fmt.Fprintln(output, "    <tbody>")

	// Обрабатываем строки результата
	for rows.Next() {
		// Создаем слайс для значений
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Сканируем значения
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		// Выводим строку таблицы
		fmt.Fprint(output, "        <tr>")
		for _, v := range values {
			var cell string
			if v == nil {
				cell = ""
			} else {
				cell = fmt.Sprintf("%v", v)
			}
			fmt.Fprintf(output, "<td>%s</td>", escapeHTML(cell))
		}
		fmt.Fprintln(output, "</tr>")
	}

	// Конец тела таблицы
	fmt.Fprintln(output, "    </tbody>")

	// Конец таблицы
	fmt.Fprintln(output, "</table>")

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	return nil
}

func executeQueryDefault(output io.Writer, config Config, columns []string, rows *sql.Rows) error {
	// Для формата Jira выводим заголовок таблицы
	if config.format == "jira" && !config.noHeader && len(columns) > 0 {
		// Выводим заголовок таблицы Jira
		fmt.Fprintln(output, "||"+strings.Join(columns, "||")+"||")
	} else if !config.noHeader && len(columns) > 0 && config.format != "jira" {
		// Для других форматов
		header := formatRow(columns, config.format)
		fmt.Fprintln(output, header)
	}

	// Обрабатываем строки результата
	for rows.Next() {
		// Создаем слайс для значений
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Сканируем значения
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		// Конвертируем значения в строки
		strValues := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				strValues[i] = ""
			} else {
				strValues[i] = fmt.Sprintf("%v", v)
			}
		}

		// Форматируем и выводим строку
		formattedRow := formatRow(strValues, config.format)
		fmt.Fprintln(output, formattedRow)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	return nil
}

func formatRow(values []string, format string) string {
	switch format {
	case "csv":
		return formatCSV(values)
	case "jira":
		return formatJira(values)
	default: // tsv
		return formatTSV(values)
	}
}

func formatTSV(values []string) string {
	return strings.Join(values, "\t")
}

func formatCSV(values []string) string {
	var result []string
	for _, v := range values {
		// Экранируем значения с запятыми, кавычками и переводами строк
		if strings.ContainsAny(v, ",\"\n") {
			// Удваиваем кавычки внутри строки
			escaped := strings.ReplaceAll(v, "\"", "\"\"")
			result = append(result, "\""+escaped+"\"")
		} else {
			result = append(result, v)
		}
	}
	return strings.Join(result, ",")
}

func formatJira(values []string) string {
	// Формат таблицы для Jira/Confluence:
	// |value1|value2|value3|

	// Экранируем символы, которые могут сломать таблицу
	escapedValues := make([]string, len(values))
	for i, v := range values {
		// Заменяем | на \|
		escaped := strings.ReplaceAll(v, "|", "\\|")
		// Заменяем переводы строк на пробелы
		escaped = strings.ReplaceAll(escaped, "\n", " ")
		escapedValues[i] = escaped
	}

	return "|" + strings.Join(escapedValues, "|") + "|"
}

func escapeHTML(s string) string {
	// Простое экранирование HTML символов
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "<")
	s = strings.ReplaceAll(s, ">", ">")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}
