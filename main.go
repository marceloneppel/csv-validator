package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// TODO: add quote and escape characters flags.
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

func validate(filePath string, delimiter string, generateValidFile bool) {
	var validFile *os.File
	var invalidFile *os.File
	var err error
	var validFileWriter *bufio.Writer
	var invalidFileWriter *bufio.Writer

	if generateValidFile {
		validFile, err = os.Create(os.Getenv("GOPATH") + "/src/github.com/marceloneppel/csv-validator/generated-files/validFile.csv")
		if err != nil {
			log.Fatal("Error: ", err.Error())
		}
		defer validFile.Close()

		validFileWriter = bufio.NewWriter(validFile)

		invalidFile, err = os.Create(os.Getenv("GOPATH") + "/src/github.com/marceloneppel/csv-validator/generated-files/invalidFile.csv")
		if err != nil {
			log.Fatal("Error: ", err.Error())
		}
		defer invalidFile.Close()

		invalidFileWriter = bufio.NewWriter(invalidFile)
	}

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

		currentLineColumnsCount, err := validateLine(line, delimiter)
		if err != nil {
			log.Println("Line number:", lineNumber, "- Error message:", err.Error(), "- Line content:", line)
			errorsCount++
			if lineNumber == 1 {
				log.Println("Aborting validation because of error on file's first line.")
				break
			}
			if generateValidFile {
				writeToFile(invalidFileWriter, line)
			}
			if errorsCount > 5 {
				break
			}
		} else if lineNumber == 1 {
			columnsCount = currentLineColumnsCount
		} else if currentLineColumnsCount != columnsCount {
			log.Println("Line number:", lineNumber, "- Error message: line with", strconv.Itoa(currentLineColumnsCount), "column(s) instead of", strconv.Itoa(columnsCount), "- Line content:", line)
			errorsCount++
			if generateValidFile {
				writeToFile(invalidFileWriter, line)
			}
		} else if generateValidFile {
			writeToFile(validFileWriter, line)
		}
	}

	if generateValidFile {
		validFileWriter.Flush()
		invalidFileWriter.Flush()
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

func validateLine(line string, delimiter string) (int, error) {
	columnsCount := 1
	var err error
	inQuote := false
	insideQuote := false
	for charIndex := 0; charIndex < len(line); charIndex++ { // Loop on line characters.
		character := string(line[charIndex])
		// TODO: add escape chars logic (https://github.com/postgres/postgres/blob/404cbc5620f4d0cec213d8804f612776dc302d55/src/backend/commands/copy.c#L3472).
		if character == "\"" {
			if charIndex == 0 || charIndex == len(line)-1 {
				inQuote = !inQuote
			} else if charIndex < len(line)-1 && string(line[charIndex+1]) == delimiter {
				isInsideQuote := false
				if insideQuote {
					for nextChar := charIndex + 2; nextChar < len(line); nextChar++ {
						if nextChar == charIndex+2 && (string(line[nextChar+1]) == "\"" || string(line[nextChar+1]) == delimiter) {
							break
						} else if string(line[nextChar]) == "\"" && nextChar < len(line)-2 && string(line[nextChar+1]) == delimiter {
							isInsideQuote = true
							break
						}
					}
					//if insideQuote && charIndex < len(line)-2 && string(line[charIndex+2]) == "\"" {
				}
				if isInsideQuote {
					insideQuote = !insideQuote
				} else {
					inQuote = !inQuote
				}
			} else if charIndex < len(line)-1 && string(line[charIndex-1]) == delimiter {
				inQuote = !inQuote
			} else {
				insideQuote = !insideQuote
			}
		}
		if character == delimiter && !inQuote {
			columnsCount++
		}
	}
	if inQuote {
		return 0, errors.New("no final quote on column")
	}
	return columnsCount, err
}

func writeToFile(writer *bufio.Writer, line string) error {
	_, err := writer.WriteString(line + "\n")
	return err
}
