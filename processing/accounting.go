package processing

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/blacked/go-zabbix"
	"github.com/golang/glog"
)

// PerfSetChan is used to transfer PerfPerfSet
var PerfSetChan = make(chan PerfSet, 100)

// PerfSet is used to send accounting datasets over the PerfSet channel
type PerfSet struct {
	Domain string
	Ident  string
	Time   string
	Code   int
}

// accountingSet for a certain request type
type accountingSet struct {
	count     int64
	lastCount int64
	sum       int64
	lastSum   int64
	codes     map[int]int64
	classes   map[int]int64
}

type zabbixConfigSetting struct {
	Server       string
	ServerPort   int
	Host         string
	DiscoveryKey string
	BaseKey      string
	Disabled     bool
}

// RequestAccounting account requests delivered by PerfSetChan
type RequestAccounting struct {
	classes           []int
	requestMappings   map[string]*regexp.Regexp
	stats             map[string]map[string]*accountingSet
	zabbixConfig      zabbixConfigSetting
	failedZabbixSends int64
}

// CompleteChan is used to wait for accounting completion
var CompleteChan = make(chan int64)

// SignalChan is used to wait for signals
var SignalChan = make(chan os.Signal, 1)

var sendMutex sync.Mutex

// NewRequestAccounting creates a RequestAccounting instance
func NewRequestAccounting(cfg Configuration) *RequestAccounting {
	// RequestAccountingInst configures the accounting
	RequestAccountingInst := RequestAccounting{
		// a list of accounting classes, defined in microseconds
		classes: cfg.ResponstimeClasses,
		// a map of requesttypes containing compiled regexes
		requestMappings: cfg.RequestMappings,
		// the current state of the statistics
		stats: map[string]map[string]*accountingSet{},
		zabbixConfig: zabbixConfigSetting{
			Server:       cfg.ZabbixServer,
			ServerPort:   10050,
			Host:         cfg.ZabbixHost,
			DiscoveryKey: "apache.discovery",
			BaseKey:      "apache.acc",
			Disabled:     cfg.ZabbixSendDisabled,
		},
	}
	go RequestAccountingInst.consumePerfSets(cfg.DiscoveryInterval, cfg.SendingInterval, cfg.Timeout)
	return &RequestAccountingInst
}

// GetFailedZabbixSends Returns the number of failed zabbix data deliveries
func (c *RequestAccounting) GetFailedZabbixSends() int64 {
	return c.failedZabbixSends
}

// SetRequestMappings defined a new set of request mappings
func (c *RequestAccounting) SetRequestMappings(mappings map[string]*regexp.Regexp) {
	c.requestMappings = mappings
}

// DisableZabbixSender Disables or Enables the submission of zabbix statistics
func (c *RequestAccounting) DisableZabbixSender(disable bool) {
	c.zabbixConfig.Disabled = disable
}

func (c *RequestAccounting) getPerfclass(responsetime int) int {
	for _, perfclass := range c.classes {
		if responsetime >= perfclass {
			return perfclass
		}
	}
	glog.Warningf("No perf class found for responstime %d", responsetime)
	return 0
}

func (c *RequestAccounting) sendDiscovery() {
	sendMutex.Lock()
	defer sendMutex.Unlock()
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
	metrics = append(metrics, NewMetric(c.zabbixConfig.Host, c.zabbixConfig.DiscoveryKey, string(jsonString), time.Now().Unix()))
	c.sendZabbixMetrics(metrics)
}

func (c *RequestAccounting) sendZabbixMetrics(metrics []*Metric) {
	if c.zabbixConfig.Disabled {
		glog.Info("Zabbix sender disabled, not sending data")
	} else {
		packet := NewPacket(metrics)
		z := NewSender(c.zabbixConfig.Server, c.zabbixConfig.ServerPort)
		res, err := z.Send(packet)
		if err != nil {
			glog.Errorf("unable to send discovery key : '%s' - >>>%s<<<", err.Error(), res)
			c.failedZabbixSends++
		}
	}
}

func (c *RequestAccounting) createZabbixMetric(dataTime int64, value string, keys ...string) *Metric {
	key := fmt.Sprintf("%s[%s]", c.zabbixConfig.BaseKey, strings.Join(keys, ","))
	glog.V(1).Infof("Creating metric : %s = %s", key, value)
	return NewMetric(c.zabbixConfig.Host, key, string(value), dataTime)
}

func (c *RequestAccounting) sendData() {
	sendMutex.Lock()
	defer sendMutex.Unlock()
	glog.Info("Sending data")
	var metrics []*Metric

	dataTime := time.Now().Unix()

	for vhost, vhostData := range c.stats {
		for accset, accsetData := range vhostData {
			metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(accsetData.count, 10), vhost, accset, "count"))
			metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(accsetData.sum, 10), vhost, accset, "sum"))

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
			metrics = append(metrics, c.createZabbixMetric(dataTime, requestsPerSecond, vhost, accset, "req_s"))

			for class, count := range accsetData.classes {
				metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "class", fmt.Sprintf("%d", class)))
			}

			for code, count := range accsetData.codes {
				metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "code", fmt.Sprintf("%d", code)))
			}
		}
	}
	c.sendZabbixMetrics(metrics)
}

// CompleteStream finishes processing
func CompleteStream() {
	PerfSetChan <- PerfSet{
		Domain: "COMPLETE",
		Ident:  "COMPLETE",
		Time:   "0",
		Code:   1,
	}
}

// ConsumePerfSets from channel PerfSetChan and send discoveries and data
func (c *RequestAccounting) consumePerfSets(discoveryIntervalSeconds int, sendingIntervalSeconds int, timeoutSeconds int) {
	var count int64 = 0
	var timeLastDiscovery time.Time = time.Now()
	var timeLastStats time.Time = time.Now()

	for {
		select {
		case signal := <-SignalChan:
			{
				glog.Infof("got %s signal, terminating myself now", signal)
				c.Showstats()
				c.sendDiscovery()
				c.sendData()
				os.Exit(1)
			}
		case perfSet := <-PerfSetChan:
			{
				if perfSet.Domain == "COMPLETE" {
					glog.Info("Processing complete")
					c.SubmitData()
					CompleteChan <- count
					return
				}
				glog.V(2).Infof("Consume a PerfSet domain: %s, ident: %s, time %s, code %d", perfSet.Domain, perfSet.Ident, perfSet.Time, perfSet.Code)
				if c.AccountRequest(perfSet.Domain, perfSet.Ident, perfSet.Time, perfSet.Code) {
					count++
				}
			}
		case <-time.After(time.Duration(timeoutSeconds) * time.Second):
			{
				glog.V(2).Infof("Timeout after %d seconds", timeoutSeconds)
			}
		}

		elapsedSecondsDataDiscovery := int(time.Since(timeLastDiscovery) / 1000000000)
		if elapsedSecondsDataDiscovery > discoveryIntervalSeconds {
			c.sendDiscovery()
			timeLastDiscovery = time.Now()
		}

		elapsedSecondsDataStats := int(time.Since(timeLastStats) / 1000000000)
		if elapsedSecondsDataStats > sendingIntervalSeconds {
			c.sendData()
			timeLastStats = time.Now()
		}

	}
}

// SubmitData delivers measures and Discovery
func (c *RequestAccounting) SubmitData() {
	c.sendDiscovery()
	c.sendData()
}

// AccountRequest accounts the request :-)
func (c *RequestAccounting) AccountRequest(domain string, uri string, time string, code int) bool {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		glog.Infof("unable to convert time '%s' to a string", time)
		return false
	}

	for name, reName := range c.requestMappings {
		match := reName.FindStringSubmatch(uri)
		if len(match) == 0 {
			continue
		}
		if (c.stats[domain] == nil) || (c.stats[domain][name] == nil) {
			c.stats[domain] = make(map[string]*accountingSet)
			c.stats[domain][name] = &accountingSet{
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
		c.stats[domain][name].classes[c.getPerfclass(responsetime)]++
	}
	return true
}

// Showstats displays the statistics
func (c *RequestAccounting) Showstats() {
	Debugit(false, "current statistics", c.stats)
}
