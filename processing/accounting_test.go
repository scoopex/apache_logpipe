package processing_test

import (
	"apache_logpipe/processing"
	"fmt"
	"testing"

	"github.com/golang/glog"
	"github.com/stretchr/testify/assert"
)

func init() {
	processing.SetupGlogForTests()
	glog.Info("Initalized Logsettings")
}

func TestSimpleRequestAccounting(t *testing.T) {
	assert := assert.New(t)
	requestAccounting := processing.NewRequestAccounting(3, 2, 1)
	requestAccounting.DisableZabbixSender(true)

	for t := 0; t < 10; t++ {
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
	assert.Equal(10, linesAccounted)

}
