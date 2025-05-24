package main

import (
	"encoding/csv"
	"os"
	"strconv"
)

func writeHeader(filename string) {
	file2, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file2.Close()

	writer := csv.NewWriter(file2)
	defer writer.Flush()
	// this defines the header value and data values for the new csv file
	headers := []string{"linesOfCode", "numberOfNodes", "numberOfCauses", "numberOfLocations", "duration"}
	writer.Write(headers)
}

func writeCsvRow(filename string, row datum) {
	// Open the CSV file for appending
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write a new row
	linesOfCode := strconv.Itoa(row.lineOfCode)
	numberOfNodes := strconv.Itoa(row.numberOfSyntaxNodes)
	numberOfCauses := strconv.Itoa(row.numberOfCauses)
	numberOfLocations := strconv.Itoa(row.numberOfLocations)
	duration := strconv.Itoa(row.duration)
	rowData := []string{linesOfCode, numberOfNodes, numberOfCauses, numberOfLocations, duration}
	err = writer.Write(rowData)
	if err != nil {
		panic(err)
	}
}

func createCsv(fileName string, data []datum) {
	file2, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}
	defer file2.Close()

	writer := csv.NewWriter(file2)
	defer writer.Flush()
	// this defines the header value and data values for the new csv file
	headers := []string{"linesOfCode", "numberOfNodes", "numberOfCauses", "numberOfLocations", "duration"}
	dataString := make([][]string, len(data))
	for i, dt := range data {
		linesOfCode := strconv.Itoa(dt.lineOfCode)
		numberOfNodes := strconv.Itoa(dt.numberOfSyntaxNodes)
		numberOfCauses := strconv.Itoa(dt.numberOfCauses)
		numberOfLocations := strconv.Itoa(dt.numberOfLocations)
		duration := strconv.Itoa(dt.duration)
		dataString[i] = []string{linesOfCode, numberOfNodes, numberOfCauses, numberOfLocations, duration}
	}
	writer.Write(headers)
	for _, row := range dataString {
		writer.Write(row)
	}

}
