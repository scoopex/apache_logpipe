package processing

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/lestrrat-go/strftime"
)

// LogSink instance manages
type LogSink struct {
	FilenamePattern         string
	SymlinkFile             string
	fileNamePatternStrftime *strftime.Strftime
	CurrentFileName         string
	fileDescriptor          *os.File
	LinesWritten            int64
}

var singletonLogSink *LogSink = nil
var mu sync.Mutex
var countChan = make(chan int64, 100)

// NewLogSink a new Logfile instance
func NewLogSink(pattern string, symlink string) *LogSink {

	// establish a lock and release lock after completing this function
	// encapsulate lock in nameless function to prevent deadlog
	// inspired by http://marcio.io/2015/07/singleton-pattern-in-go/
	func() {
		mu.Lock()
		defer mu.Unlock()
		if singletonLogSink == nil {
			singletonLogSink = new(LogSink)
		}
		singletonLogSink.FilenamePattern = pattern
		singletonLogSink.SymlinkFile = symlink
	}()
	singletonLogSink.fileDescriptor = singletonLogSink.getFileDescriptor()

	return singletonLogSink
}

func (c *LogSink) getFileDescriptor() *os.File {

	if c.fileNamePatternStrftime == nil {
		var err error
		c.fileNamePatternStrftime, err = strftime.New(c.FilenamePattern)
		if err != nil {
			glog.Fatal(err.Error())
		}
	}
	currentFilename := c.fileNamePatternStrftime.FormatString(time.Now())
	if currentFilename != c.CurrentFileName {
		var openFlags int
		if FileExists(currentFilename) {
			glog.Infof("open existing file %s for writing", currentFilename)
			openFlags = os.O_APPEND | os.O_WRONLY
		} else {
			glog.Infof("open new file %s for writing", currentFilename)
			openFlags = os.O_CREATE | os.O_WRONLY
		}

		// establish a lock and release lock after completing this function
		mu.Lock()
		defer mu.Unlock()

		f, err := os.OpenFile(currentFilename, openFlags, 0644)
		if err != nil {
			glog.Fatal(err.Error())
			return nil
		}
		if c.SymlinkFile != "" {
			if FileExists(c.SymlinkFile) {
				glog.V(1).Infof("already exists, removing existing symlink %s", c.SymlinkFile)
				os.Remove(c.SymlinkFile)
			}
			glog.V(2).Infof("creating symlink %s -> %s", currentFilename, c.SymlinkFile)
			err = os.Symlink(currentFilename, c.SymlinkFile)
			if err != nil {
				glog.Fatal(err.Error())
				return nil
			}
		}

		c.CurrentFileName = currentFilename
		c.fileDescriptor.Close()
		c.fileDescriptor = f
	} else {
		glog.V(2).Info("Reuse filedescriptor")
	}
	return c.fileDescriptor
}

// WriteLogLine writes a line to a logfile
func (c *LogSink) WriteLogLine(line string) {

	if c.FilenamePattern == "/dev/null" {
		return
	}
	c.fileDescriptor = c.getFileDescriptor()

	_, err := c.fileDescriptor.WriteString(line + "\n")
	if err != nil {
		glog.Fatal(err)
	}
	atomic.AddInt64(&c.LinesWritten, 1)
}

// CloseLogfile closes the logfile :-)
func (c *LogSink) CloseLogfile() error {
	if c.FilenamePattern == "/dev/null" {
		return nil
	}
	// establish a lock and release lock after completing this function
	mu.Lock()
	defer mu.Unlock()

	glog.V(1).Infof("closing logfile %s", c.CurrentFileName)
	var err error
	c.CurrentFileName = ""
	if c.fileDescriptor != nil {
		err = c.fileDescriptor.Close()
		c.fileDescriptor = nil
	}
	return err
}
