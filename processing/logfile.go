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
	logMessageChan          chan string
	streamStatus            chan int64
	persisterActive         bool
}

var singletonLogSink *LogSink = nil
var mu sync.Mutex

// NewLogSink a new Logfile instance
func NewLogSink(pattern string, symlink string) *LogSink {

	// establish a lock and release lock after completing this function
	// encapsulate lock in nameless function to prevent deadlog
	// inspired by http://marcio.io/2015/07/singleton-pattern-in-go/

	mu.Lock()
	defer mu.Unlock()

	if singletonLogSink == nil {
		singletonLogSink = new(LogSink)
		singletonLogSink.logMessageChan = make(chan string, 1000)
		singletonLogSink.streamStatus = make(chan int64, 1)
		go singletonLogSink.persistLogLines()
		singletonLogSink.persisterActive = true
	}

	singletonLogSink.FilenamePattern = pattern
	var err error
	singletonLogSink.fileNamePatternStrftime, err = strftime.New(singletonLogSink.FilenamePattern)
	if err != nil {
		glog.Fatal(err.Error())
	}

	singletonLogSink.SymlinkFile = symlink
	return singletonLogSink
}

func (c *LogSink) getFileDescriptor() *os.File {
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
		if c.fileDescriptor != nil {
			c.fileDescriptor.Close()
		}
		c.fileDescriptor = f
	} else {
		glog.V(2).Info("Reuse filedescriptor")
	}
	return c.fileDescriptor
}

// SubmitLogLine queues a logline for writing
func (c *LogSink) SubmitLogLine(line string) {
	c.logMessageChan <- line
}

func (c *LogSink) closeLog() {
	if c.fileDescriptor != nil {
		glog.V(1).Infof("closing logfile %s", c.CurrentFileName)
		c.fileDescriptor.Close()
		c.fileDescriptor = nil
		c.CurrentFileName = ""
	} else {
		glog.Warningf("logfile %s already closed", c.CurrentFileName)
	}
	c.streamStatus <- c.LinesWritten
}

func (c *LogSink) persistLogLines() {

	glog.Info("start persisting")
	for {
		line := <-c.logMessageChan

		c.fileDescriptor = c.getFileDescriptor()

		if line == "<END>" {
			c.closeLog()
			continue
		}

		if line == "<COMMIT>" {
			glog.Infof("commit logfile %s", c.CurrentFileName)
			c.fileDescriptor.Sync()
			c.streamStatus <- c.LinesWritten
			continue
		}

		if line == "<TERMINATE>" {
			c.closeLog()
			glog.Info("Stopping persister routine")
			return
		}

		_, err := c.fileDescriptor.WriteString(line + "\n")
		if err != nil {
			glog.Fatal(err)
		}
		atomic.AddInt64(&c.LinesWritten, 1)
	}
}

// TerminateLogStream termiates the consumer writer :-)
func (c *LogSink) TerminateLogStream() {
	mu.Lock()
	defer mu.Unlock()
	if c.persisterActive == false {
		glog.V(1).Infof("Logstream already terminated")
		return
	}
	c.SubmitLogLine("<TERMINATE>")
	nrLines := <-c.streamStatus
	glog.V(1).Infof("Stream terminated after %d lines", nrLines)

}

// CloseLogStream closes the logfile :-)
func (c *LogSink) CloseLogStream() {
	mu.Lock()
	defer mu.Unlock()
	if c.persisterActive == false {
		glog.V(1).Infof("Logstream already closed")
		return
	}
	c.SubmitLogLine("<END>")
	nrLines := <-c.streamStatus
	glog.V(1).Infof("Stream closed after %d lines", nrLines)
}

// CommitLogStream flushes the current stream to disk
func (c *LogSink) CommitLogStream() {
	mu.Lock()
	defer mu.Unlock()
	if c.persisterActive == false {
		glog.V(1).Infof("Logstream closed, commit not possible")
		return
	}
	c.SubmitLogLine("<COMMIT>")
	glog.Infof("Waiting for commit")
	nrLines := <-c.streamStatus
	glog.V(1).Infof("Stream commit after %d lines", nrLines)
}
