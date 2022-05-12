package processing_test

import (
	"256bit.org/apache_logpipe/processing"
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
	processing.PerfSetChan <- processing.PerfSet{
		Domain: "dom1",
		Ident:  "/theFoo/gag.gif",
		Time:   fmt.Sprintf("%d", 666),
		Code:   200,
	}
	processing.PerfSetChan <- processing.PerfSet{
		Domain: "dom2",
		Ident:  "/theFoo/gag.gif",
		Time:   fmt.Sprintf("%d", 661),
		Code:   200,
	}
	for t := 0; t < testDatasets; t++ {
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom1",
			Ident:  "/theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
		processing.PerfSetChan <- processing.PerfSet{
			Domain: "dom2",
			Ident:  "/theFoo",
			Time:   fmt.Sprintf("%d", t),
			Code:   200,
		}
		time.Sleep(1 * time.Second)
	}
	processing.CompleteStream()
	//time.Sleep(2 * time.Second)

	linesAccounted := <-processing.CompleteChan
	requestAccounting.SubmitData()
	assert.Equal(int64(testDatasets*2)+2, linesAccounted)

	vhosts, accountingClasses := requestAccounting.GetStatistics()
	assert.Equal(vhosts, int64(2), "search for the inserted domains")
	assert.Equal(accountingClasses, int64(4), "check if there are 4 lcasses")

	//requestAccounting.Showstats()
	requestAccounting.DumpAccountingData()
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
	requestAccounting.ShowStats()
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
	requestAccounting.ShowStats()
	requestAccounting.SubmitData()
}
