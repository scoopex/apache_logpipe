package processing_test

import (
	"apache_logpipe/processing"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func setupLogfileTestDir() string {

	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if !ok || details == nil {
		glog.Error("unable to identify caller")
		os.Exit(1)
	}

	dest := fmt.Sprintf("/tmp/%s_%d", details.Name(), rand.Intn(1000000))

	err := os.MkdirAll(dest, os.FileMode(0777))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create destination dir '%s' - %s\n", dest, err.Error())
	}
	glog.Infof("Dest Temporary Dir: %s", dest)
	return dest
}

func removeTestDir(testDir string) {
	glog.Infof("Removing testdir : %s", testDir)
	err := os.RemoveAll(testDir)
	if err != nil {
		glog.Errorf("unable to remove test dir %s : %s", testDir, err.Error())
	}
}

func init() {
	processing.SetupGlogForTests()
	glog.Info("Initalized Logsettings")
}

// https://godoc.org/github.com/stretchr/testify/assert
func TestLogfile(t *testing.T) {

	assert := assert.New(t)

	testDir := setupLogfileTestDir()
	glog.Infof("Created testdir : %s", testDir)
	defer removeTestDir(testDir)

	symlink := testDir + "/current_apache_logpipe_symlink"
	os.Symlink(testDir+"/dead-link-target", symlink)
	glog.Info("***********************************************************************")
	ls1 := processing.NewLogSink(testDir+"/apache_logpipe_test_access.log_%Y-%m-%d", symlink)
	ls1.CommitLogStream()

	filenameFirst := ls1.CurrentFileName
	assert.FileExists(filenameFirst, "file does not exist")
	assert.FileExists(symlink, "symlink does not exist")
	ls1.SubmitLogLine("TEST")
	ls1.SubmitLogLine("TEST2")
	ls1.CloseLogStream()
	assert.Equal(ls1.LinesWritten, int64(2))
	assert.Regexp(regexp.MustCompile(`/tmp/.*/apache_logpipe_test_access.log_....-..-..`), filenameFirst)

	glog.Info("***********************************************************************")
	ls2 := processing.NewLogSink(testDir+"/apache_logpipe_test_access.log_%Y-%m-%d", symlink)
	ls2.SubmitLogLine("TEST")
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
	testdir := setupLogfileTestDir()
	defer removeTestDir(testdir)
	pattern := testdir + "/apache_logpipe_test_concurrent_access.log_%Y-%m-%d"
	glog.Info(pattern)

	ls := processing.NewLogSink(pattern, "")
	ls.LinesWritten = 0
	ls.SubmitLogLine("INIT")
	theFile := ls.CurrentFileName
	glog.Info(theFile)
	ls.CloseLogStream()

	numberOfConcurrentThreads := 97
	numberOfLinesPerThread := 17

	var wg sync.WaitGroup
	for i := 0; i < numberOfConcurrentThreads; i++ {
		wg.Add(1)
		// Concurrency is currently not working
		go func(wg *sync.WaitGroup, num int) {
			//func(wg *sync.WaitGroup) {
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
	ls.CloseLogStream()
	assert.Equal(int64((numberOfConcurrentThreads*numberOfLinesPerThread*2)+1+1), ls.LinesWritten)
	ls.TerminateLogStream()
}
