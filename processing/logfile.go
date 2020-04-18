package processing

import (
	"os"
	"time"

	"github.com/golang/glog"
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
		glog.Fatalf("%s is a directory", filename)
	}
	return true
}

func getFileDescriptor() *os.File {

	if fileNamePatternStrftime == nil {
		var err error
		fileNamePatternStrftime, err = strftime.New(FilenamePattern)
		if err != nil {
			glog.Fatal(err.Error())
		}
	}

	currentFilename := fileNamePatternStrftime.FormatString(time.Now())
	if currentFilename != fileName {

		glog.Infof("open file %s for writing", currentFilename)

		var openFlags int
		if fileExists(currentFilename) {
			openFlags = os.O_APPEND | os.O_WRONLY
		} else {
			openFlags = os.O_CREATE | os.O_WRONLY
		}
		f, err := os.OpenFile(currentFilename, openFlags, 0644)
		if err != nil {
			glog.Fatal(err.Error())
			return nil
		}
		fileName = currentFilename
		fileDescriptor.Close()
		fileDescriptor = f
	} else {
		glog.V(2).Info("Reuse filedescriptor")
	}
	return fileDescriptor
}

// WriteLogLine writes a line to a logfile
func WriteLogLine(line string) {

	if FilenamePattern == "/dev/null" {
		return
	}
	fileDescriptor = getFileDescriptor()

	_, err := fileDescriptor.WriteString(line + "\n")
	if err != nil {
		glog.Info(err)
	}
}

// CloseLogfile closes the logfile :-)
func CloseLogfile() {
	if FilenamePattern == "/dev/null" {
		return
	}
	if fileDescriptor != nil {
		fileDescriptor.Close()
		fileDescriptor = nil
		fileName = ""
	}
}
