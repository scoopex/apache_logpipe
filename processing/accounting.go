package processing

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	. "github.com/blacked/go-zabbix"
	"github.com/golang/glog"
)

// AccountingSet for a certain request type
type AccountingSet struct {
	count     int64
	lastCount int64
	sum       int64
	lastSum   int64
	codes     map[int]int64
	classes   map[int]int64
}
type Accounting struct {
	classes         []int
	requestMappings map[string]*regexp.Regexp
	stats           map[string]map[string]*AccountingSet
	lastStats       map[string]map[string]*AccountingSet
}

type PerfSet struct {
	Domain string
	Ident  string
	Time   string
	Code   int
}

// RequestAccounting configures the accounting
var RequestAccounting = Accounting{
	// a list of accounting classes, defined in microseconds
	classes: []int{0, 500000, 10000000, 5000000, 60000000, 300000000},
	// a map of requesttypes containing compiled regexes
	requestMappings: map[string]*regexp.Regexp{
		"all": regexp.MustCompile(`([^?]*)\??.*`),
	},
	// the current state of the statistics
	stats:     map[string]map[string]*AccountingSet{},
	lastStats: map[string]map[string]*AccountingSet{},
}

var PerfSetChan = make(chan PerfSet, 100)
var CompleteChan = make(chan int64)
var SignalChan = make(chan os.Signal, 1)

func (c *Accounting) getPerfclasses(responsetime int) int {

	for _, perfclass := range c.classes {
		if responsetime >= perfclass {
			return perfclass
		}
	}
	return 0
}

type ZabbixConfigSetting struct {
	Server       string
	ServerPort   int
	Host         string
	DiscoveryKey string
	BaseKey      string
}

var ZabbixSender = ZabbixConfigSetting{
	Server:       "zabbix",
	ServerPort:   10050,
	Host:         GetHostname(),
	DiscoveryKey: "apache.discovery",
	BaseKey:      "apache.acc",
}

var ZabbixSenderDisabled bool = false

func (c *Accounting) sendDiscovery() {
	glog.Info("Sending discovery")

	var discoveryDataArray []map[string]string

	for vhost, vhostData := range c.stats {
		for accset := range vhostData {
			discoveryItem := map[string]string{
				"{#NAME}":   vhost,
				"{#ACCSET}": accset,
			}
			discoveryDataArray = append(discoveryDataArray, discoveryItem)
		}
	}
	jsonString, err := json.Marshal(discoveryDataArray)
	if err != nil {
		glog.Fatalf("unable to marshal json discovery data: %s", err.Error())
	}
	glog.Infof("sending discovery data >>>%s<<<", string(jsonString))
	var metrics []*Metric
	metrics = append(metrics, NewMetric(ZabbixSender.Host, ZabbixSender.DiscoveryKey, string(jsonString), time.Now().Unix()))
	sendZabbixMetrics(metrics)
}

func sendZabbixMetrics(metrics []*Metric) {

	if ZabbixSenderDisabled {
		glog.Info("Zabbix sender disabled, not sending data")
	} else {
		packet := NewPacket(metrics)
		z := NewSender(ZabbixSender.Server, ZabbixSender.ServerPort)
		res, err := z.Send(packet)
		if err != nil {
			glog.Errorf("unable to send discovery key : '%s' - >>>%s<<<", err.Error(), res)
		}
	}
}

func createZabbixMetric(dataTime int64, value string, keys ...string) *Metric {
	key := fmt.Sprintf("%s[%s]", ZabbixSender.BaseKey, strings.Join(keys, ","))
	glog.V(1).Infof("Creating metric : %s = %s", key, value)
	return NewMetric(ZabbixSender.Host, key, string(value), dataTime)
}

func (c *Accounting) sendData() {
	glog.Info("Sending data")
	var metrics []*Metric

	dataTime := time.Now().Unix()

	for vhost, vhostData := range c.stats {
		for accset, accsetData := range vhostData {
			metrics = append(metrics, createZabbixMetric(dataTime, strconv.FormatInt(accsetData.count, 10), vhost, accset, "count"))
			metrics = append(metrics, createZabbixMetric(dataTime, strconv.FormatInt(accsetData.sum, 10), vhost, accset, "sum"))

			/*
			 * Calculate differential statistics
			 */
			requestsProcessed := accsetData.count - accsetData.lastCount
			timeTaken := accsetData.sum - accsetData.lastSum
			var requestsPerSecond string = "0"
			if requestsProcessed > 0 {
				requestsPerSecond = fmt.Sprintf("%f", float64(timeTaken/requestsProcessed))
			}
			accsetData.lastCount = accsetData.count
			accsetData.lastSum = accsetData.sum
			metrics = append(metrics, createZabbixMetric(dataTime, requestsPerSecond, vhost, accset, "req_s"))

			for class, count := range accsetData.classes {
				metrics = append(metrics, createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "class", fmt.Sprintf("%d", class)))
			}

			for code, count := range accsetData.codes {
				metrics = append(metrics, createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "code", fmt.Sprintf("%d", code)))
			}
		}
	}
	sendZabbixMetrics(metrics)
}

// ConsumePerfSets from channel
func ConsumePerfSets(discoveryIntervalSeconds int, sendingIntervalSeconds int, timeoutSeconds int) {
	var count int64 = 0
	var timeLastDiscovery time.Time = time.Now()
	var timeLastStats time.Time = time.Now()

	for {
		select {
		case signal := <-SignalChan:
			{
				glog.Infof("got %s signal, terminating myself now", signal)
				RequestAccounting.Showstats()
				RequestAccounting.sendDiscovery()
				RequestAccounting.sendData()
				os.Exit(1)
			}
		case perfSet := <-PerfSetChan:
			{
				if perfSet.Domain == "COMPLETE" {
					glog.Info("Processing complete")
					RequestAccounting.sendDiscovery()
					RequestAccounting.sendData()
					CompleteChan <- count
					return
				}
				glog.V(2).Info("Consume a PerfSet")
				RequestAccounting.AccountRequest(perfSet.Domain, perfSet.Ident, perfSet.Time, perfSet.Code)
				count++
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			{
				glog.V(2).Infof("Timeout after %d seconds", timeoutSeconds)
			}
		}

		elapsedSecondsDataDiscovery := int(time.Since(timeLastDiscovery) / 1000000000)
		if elapsedSecondsDataDiscovery > discoveryIntervalSeconds {
			RequestAccounting.sendDiscovery()
			timeLastDiscovery = time.Now()
		}

		elapsedSecondsDataStats := int(time.Since(timeLastStats) / 1000000000)
		if elapsedSecondsDataStats > sendingIntervalSeconds {
			RequestAccounting.sendData()
			timeLastStats = time.Now()
		}

	}
}

// AccountRequest accounts the request :-)
func (c *Accounting) AccountRequest(domain string, uri string, time string, code int) {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		glog.Fatalf("unable to convert time '%s' to a string", time)
	}

	for name, reName := range c.requestMappings {
		match := reName.FindStringSubmatch(uri)
		if len(match) == 0 {
			continue
		}
		if (c.stats[domain] == nil) || (c.stats[domain][name] == nil) {
			c.stats[domain] = make(map[string]*AccountingSet)
			c.stats[domain][name] = &AccountingSet{
				count:   0,
				sum:     0,
				codes:   make(map[int]int64),
				classes: make(map[int]int64),
			}
			for _, perfclass := range c.classes {
				c.stats[domain][name].classes[perfclass] = 0
			}
		}

		c.stats[domain][name].sum += int64(responsetime)
		c.stats[domain][name].count++
		c.stats[domain][name].codes[code]++
		c.stats[domain][name].classes[c.getPerfclasses(responsetime)]++
	}
}

// Showstats displays the statistics
func (c *Accounting) Showstats() {
	Debugit(false, "current statistics", c.stats)
	Debugit(false, "last statistics", c.lastStats)
}
