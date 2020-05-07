package processing_test

import (
	"apache_logpipe/processing"
	"fmt"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func init() {
	SetupGlogForTests()
	glog.Info("Initalized Logsettings")
}

func TestSimpleRequestAccounting(t *testing.T) {
	assert := assert.New(t)
	requestAccounting := processing.NewRequestAccounting(3, 2, 1)
	requestAccounting.DisableZabbixSender(true)

	var testDatasets int = 10
	for t := 0; t < testDatasets; t++ {
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
	}
	processing.PerfSetChan <- processing.PerfSet{
		Domain: "COMPLETE",
		Ident:  "COMPLETE",
		Time:   "0",
		Code:   1,
	}

	linesAccounted := <-processing.CompleteChan
	assert.Equal(int64(testDatasets), linesAccounted)

}
