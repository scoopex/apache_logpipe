package processing_test

import (
	"apache_logpipe/processing"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	SetupGlogForTests()
}

func TestSimpleRequestAccounting(t *testing.T) {
	assert := assert.New(t)
	requestAccounting := processing.NewRequestAccounting(*processing.NewConfiguration())
	requestAccounting.DisableZabbixSender(true)

	var testDatasets int = 4
	for t := 0; t < testDatasets; t++ {
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
		time.Sleep(1 * time.Second)
	}
	processing.CompleteStream()
	time.Sleep(2 * time.Second)

	linesAccounted := <-processing.CompleteChan
	assert.Equal(int64(testDatasets), linesAccounted)
	requestAccounting.Showstats()
	requestAccounting.SubmitData()
}

func TestSimpleRequestAccountingWithZabbix(t *testing.T) {
	assert := assert.New(t)
	requestAccounting := processing.NewRequestAccounting(*processing.NewConfiguration())
	requestAccounting.DisableZabbixSender(false)

	var testDatasets int = 4
	for t := 0; t < testDatasets; t++ {
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
	}
	processing.CompleteStream()

	linesAccounted := <-processing.CompleteChan
	assert.Equal(int64(testDatasets), linesAccounted)
	requestAccounting.Showstats()
	requestAccounting.SubmitData()
	assert.Equal(int64(4), requestAccounting.GetFailedZabbixSends())
}

func TestBrokenData(t *testing.T) {
	assert := assert.New(t)

	requestAccounting := processing.NewRequestAccounting(*processing.NewConfiguration())

	var testDataSetLoops int64 = 4
	for t := int64(0); t < testDataSetLoops; t++ {
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "theFoo",
			Time:   fmt.Sprintf("HONK%d", t),
			Code:   200,
		}
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
	}
	processing.CompleteStream()

	linesAccounted := <-processing.CompleteChan
	assert.Equal(int64(testDataSetLoops), linesAccounted)
	requestAccounting.Showstats()
	requestAccounting.SubmitData()
}
