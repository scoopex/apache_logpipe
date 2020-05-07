package processing_test

import (
	"apache_logpipe/processing"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func init() {
	SetupGlogForTests()
}

// https://godoc.org/github.com/stretchr/testify/assert
func TestLogfile(t *testing.T) {

	assert := assert.New(t)

	testDir := SetupLogfileTestDir()
	glog.Infof("Created testdir : %s", testDir)
	defer RemoveTestDir(testDir)

	symlink := testDir + "/current_apache_logpipe_symlink"
	os.Symlink(testDir+"/dead-link-target", symlink)
	glog.Info("***********************************************************************")
	ls1 := processing.NewLogSink(testDir+"/apache_logpipe_test1_access.log_%Y-%m-%d", symlink)
	ls1.SubmitLogLine("TEST")
	ls1.CommitLogStream()
	filenameFirst := ls1.CurrentFileName
	assert.FileExists(filenameFirst, "file does not exist")
	assert.FileExists(symlink, "symlink does not exist")

	ls1.SubmitLogLine("TEST2")
	ls1.CloseLogStream()
	assert.Equal(ls1.LinesWritten, int64(2))
	assert.Regexp(regexp.MustCompile(`/tmp/.*/apache_logpipe_test1_access.log_....-..-..`), filenameFirst)

	glog.Info("***********************************************************************")
	ls2 := processing.NewLogSink(testDir+"/apache_logpipe_test1_access.log_%Y-%m-%d", symlink)
	ls2.SubmitLogLine("TEST3")
	ls2.CommitLogStream()
	assert.Equal(int64(3), ls2.LinesWritten)
	ls2.SubmitLogLine("TEST2")
	assert.Equal(&ls1, &ls2, "Adresses of pointers are not equal")
	assert.FileExists(ls2.CurrentFileName, "file does not exist")
	assert.Equal(filenameFirst, ls2.CurrentFileName, "Filenames are not equal")
	ls2.CloseLogStream()
	assert.Equal(int64(4), ls2.LinesWritten)
	glog.Info("***********************************************************************")
	ls2.TerminateLogStream()
}

func TestConcurrentLogfile(t *testing.T) {
	assert := assert.New(t)
	testdir := SetupLogfileTestDir()
	defer RemoveTestDir(testdir)
	pattern := testdir + "/apache_logpipe_test_concurrent_access.log_%Y-%m-%d"

	ls := processing.NewLogSink(pattern, "")
	ls.SubmitLogLine("INIT")
	ls.CommitLogStream()
	theFile := ls.CurrentFileName
	glog.Infof("Current logfile: %s\n", theFile)
	ls.CloseLogStream()

	numberOfConcurrentThreads := 3
	numberOfLinesPerThread := 7

	var wg sync.WaitGroup
	for i := 0; i < numberOfConcurrentThreads; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, num int) {
			defer wg.Done()
			ls = processing.NewLogSink(pattern, "")
			for t := 0; t < numberOfLinesPerThread; t++ {
				ls.SubmitLogLine(fmt.Sprintf("TEST1 - %d\n", num))
				r := rand.Intn(10)
				time.Sleep(time.Duration(r) * time.Millisecond)
				ls.SubmitLogLine(fmt.Sprintf("TEST2 - %d\n", num))
			}
			ls.CloseLogStream()
		}(&wg, i)
	}
	wg.Wait()
	ls = processing.NewLogSink(pattern, "")
	ls.SubmitLogLine("THIS IS THE END")
	ls.CommitLogStream()
	assert.Equal(int64((numberOfConcurrentThreads*numberOfLinesPerThread*2)+1+1), ls.LinesWritten)
	ls.TerminateLogStream()
}
