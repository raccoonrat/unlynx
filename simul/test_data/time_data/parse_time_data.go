package timedataunlynx

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

const spacing = 50

// addSpaces add a string with a specific number of spaces until it reaches length
func addSpaces(length int, final int) string {
	spaces := ""

	for i := 0; i < final-length; i++ {
		spaces += " "
	}
	return spaces
}

// CreateCSVFile creates and saves a CSV file
func CreateCSVFile(filename string) error {
	var fileHandle *os.File
	var err error

	fileHandle, err = os.Create(filename)
	if err != nil {
		return err
	}

	defer fileHandle.Close()
	return nil
}

// ReadTomlSetup reads the .toml and parses the different properties (e.g. Hosts)
func ReadTomlSetup(filename string, setupNbr int) (map[string]string, error) {
	var parameters []string

	setup := make(map[string]string)

	fileHandle, err := os.Open(filename)

	if err != nil {
		return nil, err
	}
	defer fileHandle.Close()

	scanner := bufio.NewScanner(fileHandle)

	flag := false
	pos := 0
	for scanner.Scan() {
		line := scanner.Text()

		c := strings.Split(line, ", ")

		if flag == true {
			if pos == setupNbr {
				for i, el := range c {
					setup[parameters[i]] = el
				}
				break
			}
			pos++
		}

		if c[0] == "Hosts" {
			flag = true
			parameters = c
		}

	}

	return setup, nil
}

// WriteDataFromCSVFile gets the flags and the time values (parsed from the CSV file) and writes everything into a nice .txt file
func WriteDataFromCSVFile(filename string, flags []string, testTimeData map[string][]string, pos int, setup map[string]string) error {

	var fileHandle *os.File
	var err error

	fileHandle, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)

	if err != nil {
		return err
	}
	defer fileHandle.Close()

	writer := bufio.NewWriter(fileHandle)

	_, err = fileHandle.WriteString("\n\n\n|-------------------------------------------------------------------------|\n" +
		"|----------------------------- SIMULATION #" + fmt.Sprintf("%v", pos+1) + " -----------------------------|\n" +
		"|-------------------------------------------------------------------------|\n\n\n")

	if err != nil {
		return err
	}

	for k, v := range setup {
		_, err = fileHandle.WriteString(k + ":" + addSpaces(len(k), spacing) + v + "\n")

		if err != nil {
			return err
		}
	}
	_, err = fileHandle.WriteString("\n")

	if err != nil {
		return err
	}

	for _, value := range flags {
		var err error

		if value != "\n" {
			if len(testTimeData[value]) > 0 {
				_, err = fileHandle.WriteString(value + ":" + addSpaces(len(value), spacing) + testTimeData[value][pos] + "\n")
			} else {
				_, err = fileHandle.WriteString(value + ":" + addSpaces(len(value), spacing) + "\n")
			}
		} else {
			_, err = fileHandle.WriteString(value)
		}

		if err != nil {
			return err
		}
	}

	defer writer.Flush()
	return nil
}

// ReadDataFromCSVFile reads data from the CSV file where the time values are stored and re-arranges everything in a key-value map
func ReadDataFromCSVFile(filename string, flags []string) (map[string][]string, error) {
	fileHandle, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fileHandle.Close()

	lines, err := csv.NewReader(fileHandle).ReadAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)

	for line := 1; line < len(lines); line++ {

		for i, l := range lines[line] {

			s := strings.Split(lines[0][i], "_")

			for _, el := range s {

				if stringInSlice(el, flags) {
					if len(s) >= 2 {
						if s[len(s)-1] == "sum" && s[len(s)-2] == "wall" { //Only the time values that have wall in the end matter
							if _, ok := result[el]; ok && len(result[el]) == line {
								result[el][line-1] += ", " + l
							} else {
								result[el] = append(result[el], l)
							}
							continue
						}
					} else {
						result[el] = append(result[el], l)
						continue
					}
				}
			}
		}
	}

	return result, nil
}

// stringInSlice checks if a string is inside an array of strings
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
