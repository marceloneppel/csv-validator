package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func escapeQuotes(internalString string) string {
	var regex = regexp.MustCompile(`(?:\")`)
	/*ocurrences := regex.FindAllString(internalString, -1)
	if len(ocurrences)%2 == 0 {*/
	matchesIndexes := regex.FindAllStringSubmatchIndex(internalString, -1)
	matchIndex := -1
	replacedString := regex.ReplaceAllStringFunc(internalString, func(match string) string {
		matchIndex++
		log.Println("matchIndex:", matchIndex, "match:", match, matchesIndexes)
		matchStartIndex := matchesIndexes[matchIndex][0]
		matchEndIndex := matchesIndexes[matchIndex][1]
		if matchStartIndex == 0 {
			return "\"" + match
		}
		if matchStartIndex > 0 && internalString[matchStartIndex-1:matchEndIndex-1] != "\"" {
			return "\"" + match
		}
		return match
	})
	log.Println("field replaced:", replacedString)
	return replacedString
	//}
	//return internalString
}

func main() {
	args := os.Args
	if len(args) < 3 {
		log.Fatal("Too few arguments.")
	}
	if len(args) > 4 {
		log.Fatal("Too many arguments.")
	}
	filePath := args[1:2][0]
	fmt.Println("filePath:", filePath)
	if _, err := os.Stat(filePath); err == nil {
		delimiter := args[2:3][0]
		generateValidFileParameter := "false"
		if len(args) == 4 {
			generateValidFileParameter = args[3:4][0]
		}
		if len(delimiter) != 1 {
			log.Fatal("Invalid delimiter.")
		}
		if generateValidFileParameter != "false" && generateValidFileParameter != "true" {
			log.Fatal("Invalid value for generateValidFile parameter (valid values: false, true).")
		}
		generateValidFile := false
		if generateValidFileParameter == "true" {
			generateValidFile = true
		}
		log.Println("Processing file", filePath, "with delimiter", delimiter)
		validate(filePath, delimiter, generateValidFile)
	} else if os.IsNotExist(err) {
		log.Fatal("File ", filePath, " doesn't exist.")
	} else {
		log.Fatal("Error: ", err.Error())
	}
}

// Postgres behavior.
func parseFields(parseError csv.ParseError, line string, delimiter string, columnsCount int) (*csv.ParseError, int) {
	log.Println("")
	err := &parseError
	fields := strings.Split(line, delimiter)
	log.Println("line:", line)
	log.Println("fields:", fields)
	for fieldIndex, field := range fields {
		log.Println("field:", field)
		if err.Err == csv.ErrQuote {
			if len(field) > 1 && strings.HasPrefix(field, "\"") && strings.HasSuffix(field, "\"") {
				mergeFields := false
				for nextIndex := fieldIndex + 1; nextIndex < len(fields); nextIndex++ {
					if !strings.HasPrefix(fields[fieldIndex+1], "\"") && strings.HasSuffix(fields[fieldIndex+1], "\"") {
						mergeFields = true
						break
					}
				}
				if mergeFields {
					log.Println("joinFieldIndex:", fieldIndex, "- escapedJoinField:", escapeQuotes(field[1:len(field)]))
					fields[fieldIndex] = "\"" + escapeQuotes(field[1:len(field)])
					//} else if len(fields) > fieldIndex+1 && len(fields[fieldIndex+1]) > 1 && !strings.HasPrefix(fields[fieldIndex+1], "\"") && strings.HasSuffix(fields[fieldIndex+1], "\"") {

				} else {
					internalString := field[1 : len(field)-1]
					log.Println("fieldIndex:", fieldIndex, "- field:", internalString)
					fields[fieldIndex] = "\"" + escapeQuotes(internalString) + "\""
				}
			} else if len(field) > 1 && strings.HasPrefix(field, "\"") && len(fields) > fieldIndex+1 && len(fields[fieldIndex+1]) > 1 && strings.HasSuffix(fields[fieldIndex+1], "\"") {
				log.Println("joinFieldIndex:", fieldIndex, "- escapedJoinField:", escapeQuotes(field[1:len(field)-1]))
				fields[fieldIndex] = "\"" + escapeQuotes(field[1:len(field)-1])
			}
		}
	}
	if err.Err == csv.ErrQuote {
		newLine := strings.Join(fields, delimiter)
		log.Println("newLine:", newLine)
		reader := csv.NewReader(strings.NewReader(newLine))
		reader.Comma = []rune(delimiter)[0]
		parsedLine, secondParseError := reader.Read()
		newColumnsCount := len(parsedLine)
		if secondParseError != nil && secondParseError.(*csv.ParseError).Err != nil {
			log.Println("error2:", secondParseError.(*csv.ParseError).Error())
			return secondParseError.(*csv.ParseError), newColumnsCount
		}
		return &csv.ParseError{}, newColumnsCount
	}
	return err, columnsCount
}

func validate(filePath string, delimiter string, generateValidFile bool) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}

	errorsCount := 0

	fileReader := bufio.NewReader(csvFile)
	lineNumber := 0
	columnsCount := 0
	for {
		lineNumber++
		line, err := fileReader.ReadString('\n')
		if len(line) == 0 && err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error: ", err.Error())
		}
		line = strings.TrimSuffix(line, "\n")

		reader := csv.NewReader(strings.NewReader(line))
		reader.Comma = []rune(delimiter)[0]
		parsedLine, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			parseError := err.(*csv.ParseError)
			parseError, currentLineColumnsCount := parseFields(*parseError, line, delimiter, len(parsedLine))
			if parseError.Err != nil {
				log.Println("Line number:", lineNumber, "- Error message:", parseError.Err.Error(), "- Line content:", line)
				errorsCount++
				if lineNumber == 1 {
					log.Println("Aborting validation because of error on file's first line.")
					break
				}
				break
			}
			if currentLineColumnsCount != columnsCount {
				log.Println("Line number:", lineNumber, "- Error message: line with", strconv.Itoa(currentLineColumnsCount), "column(s) instead of", strconv.Itoa(columnsCount), "- Line content:", line)
				errorsCount++
				break
			}
			//break
		} else if lineNumber == 1 {
			columnsCount = len(parsedLine)
		} else if len(parsedLine) != columnsCount {
			_, currentLineColumnsCount := parseFields(csv.ParseError{}, line, delimiter, len(parsedLine))
			if currentLineColumnsCount != columnsCount {
				log.Println("Line number:", lineNumber, "- Error message: line with", strconv.Itoa(currentLineColumnsCount), "column(s) instead of", strconv.Itoa(columnsCount), "- Line content:", line)
				errorsCount++
			}
		}
	}

	switch errorsCount {
	case 0:
		log.Println("No errors found.")
		break
	case 1:
		log.Println("1 error found.")
		break
	default:
		log.Println(errorsCount, "errors found.")
		break
	}
}
