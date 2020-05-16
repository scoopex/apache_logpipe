package main

import (
	"apache_logpipe/processing"
	"bufio"
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
)

var helpFlag bool
var verboseFlag bool
var outputLogfile string
var sendingInterval int
var discoveryInterval int
var zabbixServer string
var zabbixHost string

func parseInput(logSink processing.LogSink, requestAccounting processing.RequestAccounting, cfg processing.Configuration) {

	scanner := bufio.NewScanner(os.Stdin)

	var lines int64 = 0
	var linesNotMatched int64 = 0
	timeStart := time.Now()

	for scanner.Scan() {
		line := scanner.Text()
		lines++
		logSink.SubmitLogLine(line)
		match := cfg.RegexLogline.FindStringSubmatch(line)
		if len(match) == 0 {
			glog.V(1).Infof("not matched line: %s\n", line)
			linesNotMatched++
			continue
		}
		result := make(map[string]string)
		for i, name := range cfg.RegexLogline.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		code, err := strconv.Atoi(result["code"])
		if err != nil {
			glog.Fatalf("unable to convert code '%s' to a string", result["code"])
		}
		if code >= 400 || code < 200 {
			linesNotMatched++
			continue
		}
		matchStatic := cfg.RegexStaticContent.FindStringSubmatch(result["uri"])
		if len(matchStatic) == 0 {
			processing.PerfSetChan <- processing.PerfSet{
				Domain: result["domain"],
				Ident:  result["uri"],
				Time:   result["time"],
				Code:   code,
			}
		} else {
			processing.PerfSetChan <- processing.PerfSet{
				Domain: result["domain"],
				Ident:  "NOT MATCHED",
				Time:   result["time"],
				Code:   code,
			}
		}
	}
	linesAccounted := logSink.CloseLogStream()
	//linesAccounted := <-processing.CompleteChan
	glog.V(1).Infof("Accounted %d lines", linesAccounted)
	if linesAccounted != lines-linesNotMatched {
		glog.Errorf("Accounted lines are not equal to matched lines (total lines: %d, lines not matched: %d, lines accounted: %d)",
			lines, linesNotMatched, linesAccounted)
	}

	elapsed := time.Since(timeStart)
	linesPerSecond := float64(lines) / (float64(elapsed) / 1000000000)
	percentageNotMatched := (float64(linesNotMatched) / float64(lines)) * 100
	glog.Infof("Processed %d lines in %s, %f lines per second, %d lines not matched (%0.2f%%)\n", lines, elapsed, linesPerSecond, linesNotMatched, percentageNotMatched)
	requestAccounting.Showstats()
}

func main() {

	cfg := processing.NewConfiguration()

	/*
	* Parsing the arguments
	 */
	configFile := *flag.String("config", "", "Name of the config file")
	cfg.OutputLogfile = *flag.String("output_logfile", cfg.OutputLogfile, "Filename with timestamp, i.e. '/var/log/apache2/access.log.%Y-%m-%d'")
	cfg.OutputLogfileSymlink = *flag.String("symlink", cfg.OutputLogfileSymlink, "A symlink which points to the current logfile")
	cfg.SendingInterval = *flag.Int("sending_interval", cfg.SendingInterval, "Sending interval in seconds")
	cfg.Timeout = *flag.Int("timeout", cfg.Timeout, "timeout in seconds (default: 5 seconds)")
	cfg.DiscoveryInterval = *flag.Int("discovery_interval", cfg.DiscoveryInterval, "Discovery interval in seconds")
	cfg.ZabbixServer = *flag.String("zabbix_server", cfg.ZabbixServer, "The hostname of the zabbix server")
	cfg.ZabbixHost = *flag.String("zabbix_host", cfg.ZabbixHost, "The zabbix host to report data for")
	cfg.ZabbixSendDisabled = *flag.Bool("disable_zabbix", cfg.ZabbixSendDisabled, "Disable zabbix sender")

	flag.Set("logtostderr", "true")
	flag.Parse()
	cfg.LoadFile(configFile)

	glog.Infof("Starting apache_logpipe: output_logfile: %s, sending_interval: %d, discovery_interval: %d, zabbix_server: %s, zabbix_host: %s\n",
		cfg.OutputLogfile, cfg.SendingInterval, cfg.DiscoveryInterval, cfg.ZabbixServer, cfg.ZabbixHost)

	// Install signal handler
	signal.Notify(processing.SignalChan, syscall.SIGINT, syscall.SIGTERM)

	logSink := *processing.NewLogSink(cfg.OutputLogfile, cfg.OutputLogfileSymlink)

	requestAccounting := processing.NewRequestAccounting(cfg.DiscoveryInterval, cfg.SendingInterval, cfg.Timeout)
	requestAccounting.DisableZabbixSender(cfg.ZabbixSendDisabled)
	parseInput(logSink, *requestAccounting, *cfg)
}
