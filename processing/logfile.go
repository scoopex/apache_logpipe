package processing

import (
	"log"
	"os"
	"time"

	"github.com/lestrrat-go/strftime"
)

// https://www.joeshaw.org/dont-defer-close-on-writable-files/

// FilenamePattern defines a pattern for the logfile using date patterns
var FilenamePattern string = ""
var fileNamePatternStrftime *strftime.Strftime = nil
var fileName string = ""
var fileDescriptor *os.File

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if info.IsDir() {
		log.Fatalf("%s is a directory", filename)
	}
	return true
}

func getFileDescriptor() *os.File {

	if fileNamePatternStrftime == nil {
		var err error
		fileNamePatternStrftime, err = strftime.New(FilenamePattern)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	currentFilename := fileNamePatternStrftime.FormatString(time.Now())
	if currentFilename != fileName {

		log.Printf("open file %s for writing", currentFilename)

		var openFlags int
		if fileExists(currentFilename) {
			openFlags = os.O_APPEND | os.O_WRONLY
		} else {
			openFlags = os.O_CREATE | os.O_WRONLY
		}
		f, err := os.OpenFile(currentFilename, openFlags, 0644)
		if err != nil {
			log.Fatal(err.Error())
			return nil
		}
		fileName = currentFilename
		fileDescriptor.Close()
		fileDescriptor = f
	} else {
		log.Println("Reuse filedescriptor")
	}
	return fileDescriptor
}

// WriteLogLine writes a line to a logfile
func WriteLogLine(line string) {

	fileDescriptor = getFileDescriptor()

	_, err := fileDescriptor.WriteString(line + "\n")
	if err != nil {
		log.Println(err)
	}
}

// CloseLogfile closes the logfile :-)
func CloseLogfile() {
	if fileDescriptor != nil {
		fileDescriptor.Close()
		fileDescriptor = nil
		fileName = ""
	}
}
