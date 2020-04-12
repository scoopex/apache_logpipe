package processing_test

import (
	"apache_logpipe/processing"
	"testing"
	"time"
)

func TestLogfile(t *testing.T) {
	processing.FilenamePattern = "/tmp/apache_logpipe_test_access.log_%Y-%m-%d"
	processing.WriteLogLine("TEST")
	time.Sleep(1000)
	processing.WriteLogLine("TEST2")
	processing.CloseLogfile()
}
