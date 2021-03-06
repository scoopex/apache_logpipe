package processing

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	. "github.com/blacked/go-zabbix"
	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"
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
	Count     int64
	lastCount int64
	Sum       int64
	lastSum   int64
	Codes     map[int]int64
	Classes   map[int]int64
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
	classes            []int
	requestMappings    map[string]*regexp.Regexp
	regexStaticContent *regexp.Regexp
	stats              map[string]map[string]*accountingSet
	zabbixConfig       zabbixConfigSetting
	failedZabbixSends  int64
	fractionOfSecond   int
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
		requestMappings:    cfg.RequestMappings,
		regexStaticContent: regexp.MustCompile(cfg.RegexStaticContentString),
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
	if c.zabbixConfig.Disabled {
		glog.V(1).Info("Zabbix sender disabled, not sending data")
		return
	}
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
	if c.zabbixConfig.Disabled {
		glog.V(1).Info("Zabbix sender disabled, not sending data")
		return
	}
	glog.Info("Sending data")
	var metrics []*Metric

	dataTime := time.Now().Unix()

	for vhost, vhostData := range c.stats {
		for accset, accsetData := range vhostData {
			metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(accsetData.Count, 10), vhost, accset, "count"))
			metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(accsetData.Sum, 10), vhost, accset, "sum"))

			/*
			 * Calculate differential statistics
			 */
			requestsProcessed := accsetData.Count - accsetData.lastCount
			timeTaken := accsetData.Sum - accsetData.lastSum
			var requestsPerSecond string = "0"
			if requestsProcessed > 0 {
				requestsPerSecond = fmt.Sprintf("%f", float64(timeTaken/requestsProcessed))
			}
			accsetData.lastCount = accsetData.Count
			accsetData.lastSum = accsetData.Sum
			metrics = append(metrics, c.createZabbixMetric(dataTime, requestsPerSecond, vhost, accset, "req_s"))

			for class, count := range accsetData.Classes {
				metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "class", fmt.Sprintf("%d", class)))
			}

			for code, count := range accsetData.Codes {
				metrics = append(metrics, c.createZabbixMetric(dataTime, strconv.FormatInt(count, 10), vhost, accset, "code", fmt.Sprintf("%d", code)))
			}
		}
	}
	c.sendZabbixMetrics(metrics)
}

func (c *RequestAccounting) collectCodes() []int {

	codes := map[int]int{}
	for _, vhostData := range c.stats {
		for _, accsetData := range vhostData {
			for code := range accsetData.Codes {
				codes[code]++
			}
		}
	}
	result := []int{}
	for code := range codes {
		result = append(result, code)
	}

	sort.Ints(result)
	return result
}

// DumpAccountingData dumps the accounting data
func (c *RequestAccounting) DumpAccountingData() {
	sendMutex.Lock()
	defer sendMutex.Unlock()

	table := tablewriter.NewWriter(os.Stdout)

	header := []string{"Domain", "PerfClass", "Count", "Average ms"}

	fmt.Printf("\n")
	for _, perfClass := range c.classes {
		header = append(header, fmt.Sprintf(" >=\n%d\nmSec", perfClass/1000))
	}
	codes := c.collectCodes()
	for _, code := range codes {
		header = append(header, fmt.Sprintf("HTTP\n%d", code))
	}
	//table.SetHeader(header1)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: true})
	table.SetHeader(header)
	table.SetHeaderAlignment(tablewriter.ALIGN_RIGHT)
	table.SetAutoFormatHeaders(false)
	for vhost, vhostData := range c.stats {
		for accset, accsetData := range vhostData {
			averageTime := float64(accsetData.Sum) / float64(accsetData.Count)
			row := []string{vhost, accset, strconv.FormatInt(accsetData.Count, 10), fmt.Sprintf("%.03f", averageTime)}
			for class := range c.classes {
				row = append(row, strconv.FormatInt(accsetData.Classes[class], 10))
			}

			for code := range codes {
				row = append(row, strconv.FormatInt(accsetData.Codes[code], 10))
			}
			table.Append(row)
		}
	}
	table.Render()
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

func (c *RequestAccounting) addAccounting(domain string, ident string, responsetime int, code int) bool {
	if c.stats[domain] == nil {
		c.stats[domain] = make(map[string]*accountingSet)
	}
	if c.stats[domain][ident] == nil {
		c.stats[domain][ident] = &accountingSet{
			Count:   0,
			Sum:     0,
			Codes:   make(map[int]int64),
			Classes: make(map[int]int64),
		}
		for _, perfclass := range c.classes {
			c.stats[domain][ident].Classes[perfclass] = 0
		}
	}
	c.stats[domain][ident].Sum += int64(responsetime)
	c.stats[domain][ident].Count++
	c.stats[domain][ident].Codes[code]++
	c.stats[domain][ident].Classes[c.getPerfclass(responsetime)]++
	return true
}

// AccountRequest accounts the request :-)
func (c *RequestAccounting) AccountRequest(domain string, uri string, time string, code int) bool {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		glog.Infof("unable to convert time '%s' to a string", time)
		return false
	}
	matchStatic := c.regexStaticContent.FindStringSubmatch(uri)
	if len(matchStatic) != 0 {
		c.addAccounting(domain, "NOT MATCHED", responsetime, code)
		return true
	}
	for name, reName := range c.requestMappings {
		match := reName.FindStringSubmatch(uri)
		if len(match) == 0 {
			continue
		}
		c.addAccounting(domain, name, responsetime, code)
	}
	return true
}

// ShowStats displays the statistics
func (c *RequestAccounting) ShowStats() {
	Debugit(false, "current statistics", c.stats)
}

// ShowStats displays the statistics
func (c *RequestAccounting) GetJsonStats() string {

	jsonString, err := json.MarshalIndent(c.stats, "", " ")
	if err != nil {
		glog.Fatalf("unable to marshal json stats data: %s", err.Error())
	}
	return string(jsonString)
}

// GetStatistics for Testcasess
func (c *RequestAccounting) GetStatistics() (int64, int64) {
	var vhosts int64 = 0
	var accountedClasses int64 = 0
	for _, vhostData := range c.stats {
		vhosts++
		for range vhostData {
			accountedClasses++
		}
	}
	return vhosts, accountedClasses
}
