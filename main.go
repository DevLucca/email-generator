package main

import (
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/osteele/liquid"
	"github.com/xuri/excelize/v2"
)

//go:embed template.html
var s string

var (
	filename  *string
	lineNum   uint8
	liquidEng *liquid.Engine
)

var outputPath = fmt.Sprintf("%s/Desktop/email-output", os.Getenv("HOME"))

var validFileExtensions = map[string]func(filename string) (rows []row, err error){
	"csv":  loadCSV,
	"xlsx": loadXLSX,
}

func init() {
	err := os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		fmt.Println(fmt.Errorf("um erro ocorreu: %s", err))
	}

	filename = flag.String("file", "", "Nome do arquivo a ser utilizado")
	flag.Parse()

	if *filename == "" {
		defer flag.Usage()
		fmt.Println("arquivo não informado.")
	}

	liquidEng = liquid.NewEngine()
}

type row struct {
	Name     string
	Doctor   string
	Date     time.Time
	Invoice  string
	BankSlip string
}

func main() {
	fileNameInfo := strings.Split(*filename, ".")
	fileExtension := fileNameInfo[len(fileNameInfo)-1]

	if handler, ok := validFileExtensions[fileExtension]; ok {
		rows, err := handler(*filename)
		if err != nil {
			panic(err)
		}
		generateOutput(rows)
	} else {
		panic(fmt.Errorf("extensão de arquivo inválida: %s", fileExtension))
	}
}

func loadCSV(filename string) (rows []row, err error) {

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo: %s", err)
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()

	for _, line := range records[1:] {
		row, err := validateRow(line)
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func loadXLSX(filename string) (rows []row, err error) {

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, err
	}

	lines, err := file.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, line := range lines[1:] {
		row, err := validateRow(line)
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func validateRow(record []string) (row row, err error) {
	const dateFormat = "2/1/2006"

	if record[0] == "" {
		return row, fmt.Errorf("nome não informado na linha: %d", lineNum+1)
	}
	row.Name = record[0]

	row.Date, err = time.Parse(dateFormat, record[1])
	if err != nil {
		return row, fmt.Errorf("data inválida na linha: %d", lineNum+1)
	}

	if record[2] == "" {
		return row, fmt.Errorf("nome do médico não informado na linha: %d", lineNum+1)
	}
	row.Doctor = record[2]

	if record[3] == "" {
		return row, fmt.Errorf("link da nota fiscal não informado na linha: %d", lineNum+1)
	}
	row.Invoice = record[3]

	if record[4] == "" {
		return row, fmt.Errorf("link do boleto não informado na linha: %d", lineNum+1)
	}
	row.BankSlip = record[4]

	lineNum++
	return
}

func generateOutput(rows []row) {
	for _, row := range rows {
		liquidifyRow(row)
	}
}

func liquidifyRow(row row) (err error) {
	bindings := map[string]interface{}{
		"name":     row.Name,
		"date":     row.Date,
		"doctor":   row.Doctor,
		"invoice":  row.Invoice,
		"bankSlip": row.BankSlip,
	}
	out, err := liquidEng.ParseAndRenderString(s, bindings)
	if err != nil {
		return err
	}
	f, err := os.Create(fmt.Sprintf("%s/%s.html", outputPath, row.Name))
	if err != nil {
		return err
	}
	f.Write([]byte(out))
	return nil
}
