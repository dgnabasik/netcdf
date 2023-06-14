package main

// Counts missing values in CSV file.
import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadTextLines(filePath string, normalizeText bool) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ReadTextLines: %+v\n", err)
		return nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		fmt.Printf("ReadTextLines: %+v\n", err)
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	maxCapacity := fi.Size() + 1
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, int(maxCapacity))

	for scanner.Scan() {
		if normalizeText {
			str := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if len(str) > 0 {
				lines = append(lines, str)
			}
		} else {
			lines = append(lines, scanner.Text())
		}
	}
	fmt.Println("Read from file " + filePath)
	return lines, scanner.Err()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Specify path to CSV file.")
		os.Exit(1)
	}
	lastRow := 0
	if len(os.Args) > 2 {
		lastRow, _ = strconv.Atoi(os.Args[2])
	}
	lines, _ := ReadTextLines(os.Args[1], false)
	columnNames := strings.Split(lines[0], ",")
	columnValues := make(map[int]int, 0)
	for ndx := 0; ndx < len(columnNames); ndx++ {
		columnValues[ndx] = 0
	}
	if lastRow == 0 {
		lastRow = len(lines)
	}
	for ndx := 1; ndx < lastRow; ndx++ {
		tokens := strings.Split(lines[ndx], ",")
		for ndx1 := 0; ndx1 < len(columnNames); ndx1++ {
			if len(tokens[ndx1]) > 0 {
				columnValues[ndx1]++
			}
		}
	}
	for ndx1 := 0; ndx1 < len(columnNames); ndx1++ {
		fmt.Print(columnNames[ndx1] + "  ")
		fmt.Println(columnValues[ndx1])
	}
}
