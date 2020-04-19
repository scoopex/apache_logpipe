package processing_test

import (
	"apache_logpipe/processing"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

// https://godoc.org/github.com/stretchr/testify/assert
func TestLogfile(t *testing.T) {
	assert := assert.New(t)
	processing.SetupGlogForTests()
	symlink := "/tmp/apache_logpipe_symlink"
	if processing.FileExists(symlink) {
		os.Remove(symlink)
	}
	os.Symlink("/tmp/dead-link", symlink)

	ls1 := processing.NewLogSink("/tmp/apache_logpipe_test_access.log_%Y-%m-%d", symlink)
	assert.FileExists(ls1.CurrentFileName, "file does not exist")
	assert.FileExists(symlink, "symlink does not exist")
	ls1.WriteLogLine("TEST")
	ls1.WriteLogLine("TEST2")
	assert.Equal(ls1.LinesWritten, int64(2))
	filenameFirst := ls1.CurrentFileName
	ls1.CloseLogfile()
	assert.Regexp(regexp.MustCompile(`/tmp/apache_logpipe_test_access.log_....-..-..`), filenameFirst)

	glog.Info("***********************************************************************")
	ls2 := processing.NewLogSink("/tmp/apache_logpipe_test_access.log_%Y-%m-%d", symlink)
	glog.Info(ls2.CurrentFileName)
	assert.Equal(&ls1, &ls2, "Adresses of pointers are not equal")
	assert.FileExists(ls2.CurrentFileName, "file does not exist")
	ls2.WriteLogLine("TEST")
	ls2.WriteLogLine("TEST2")
	assert.Equal(int64(4), ls2.LinesWritten)
	assert.Equal(filenameFirst, ls2.CurrentFileName, "Filenames are not equal")
	ls2.CloseLogfile()
}

func TestConcurrentLogfile(t *testing.T) {
	assert := assert.New(t)
	pattern := "/tmp/apache_logpipe_test_concurrent_access.log_%Y-%m-%d"
	//processing.SetupGlogForTests()

	ls := processing.NewLogSink(pattern, "")
	ls.LinesWritten = 0
	theFile := ls.CurrentFileName
	if processing.FileExists(theFile) {
		os.Remove(theFile)
	}
	ls.CloseLogfile()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		// Concurrency is currently not working
		//go func(wg *sync.WaitGroup) {
		func(wg *sync.WaitGroup) {
			defer wg.Done()
			ls = processing.NewLogSink(pattern, "")
			ls.WriteLogLine("TEST1")
			ls.WriteLogLine("TEST1")
			ls.CloseLogfile()
		}(&wg)
	}
	wg.Wait()
	ls = processing.NewLogSink(pattern, "")

	assert.Equal(int64(20), ls.LinesWritten)
}
