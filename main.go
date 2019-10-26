package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		log.Fatal("Too few arguments.")
	}
	if len(args) > 3 {
		log.Fatal("Too many arguments.")
	}
	filePath := args[1:2][0]
	delimiter := args[2:3][0]
	if len(delimiter) != 1 {
		log.Fatal("Invalid delimiter.")
	}
	fmt.Println("filePath:", filePath)
	if _, err := os.Stat(filePath); err == nil {
		log.Println("Processing file", filePath, "with delimiter", delimiter)
		validate(filePath, delimiter)
	} else if os.IsNotExist(err) {
		log.Fatal("File ", filePath, " doesn't exist.")
	} else {
		log.Fatal("Error: ", err.Error())
	}
}

func validate(filePath string, delimiter string) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Error: ", err.Error())
	}

	errorsCount := 0

	fileReader := bufio.NewReader(csvFile)
	lineNumber := 0
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
		_, err = reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			parseError := err.(*csv.ParseError)
			log.Println("Line number:", lineNumber, "- Error message:", parseError.Err.Error(), "- Line content:", line)
			errorsCount++
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
