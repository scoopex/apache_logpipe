package main

import (
	"apache_logpipe/processing"
	"bufio"
	goflag "flag"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
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
	lineRe := regexp.MustCompile(cfg.RegexLogLineString)

	for scanner.Scan() {
		line := scanner.Text()
		lines++
		logSink.SubmitLogLine(line)
		match := lineRe.FindStringSubmatch(line)
		if len(match) == 0 {
			glog.V(1).Infof("not matched line: %s\n", line)
			linesNotMatched++
			continue
		}
		result := make(map[string]string)
		for i, name := range lineRe.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}

		code, err := strconv.Atoi(result["code"])
		if err != nil {
			glog.Fatalf("unable to convert code '%s' to integer", result["code"])
		}
		if code >= 400 || code < 200 {
			linesNotMatched++
			continue
		}

		processing.PerfSetChan <- processing.PerfSet{
			Domain: result["domain"],
			Ident:  result["uri"],
			Time:   result["time"],
			Code:   code,
		}
	}
	linesAccounted := logSink.CloseLogStream()
	glog.V(1).Infof("Accounted %d lines", linesAccounted)
	if linesAccounted != lines-linesNotMatched {
		glog.Errorf("Accounted lines are not equal to matched lines (total lines: %d, lines not matched: %d ( 200 < http code, http code  >=400), lines accounted: %d)",
			lines, linesNotMatched, linesAccounted)
	}

	elapsed := time.Since(timeStart)
	linesPerSecond := float64(lines) / (float64(elapsed) / 1000000000)
	percentageNotMatched := (float64(linesNotMatched) / float64(lines)) * 100
	glog.Infof("Processed %d lines in %s, %f lines per second, %d lines not matched (%0.2f%%)\n", lines, elapsed, linesPerSecond, linesNotMatched, percentageNotMatched)
}

func main() {

	cfg := processing.NewConfiguration()

	/*
	* Parsing the arguments
	 */
	var configFile string = ""
	var showStats bool = false
	var dumpStats bool = false
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	flag.StringVar(&configFile, "config", configFile, "Name of the config file")
	flag.StringVar(&cfg.OutputLogfile, "output_logfile", cfg.OutputLogfile, "Filename with timestamp, i.e. '/var/log/apache2/access.log.%Y-%m-%d'")
	flag.StringVar(&cfg.OutputLogfileSymlink, "symlink", cfg.OutputLogfileSymlink, "A symlink which points to the current logfile")
	flag.IntVar(&cfg.SendingInterval, "sending_interval", cfg.SendingInterval, "Sending interval in seconds")
	flag.IntVar(&cfg.Timeout, "timeout", cfg.Timeout, "timeout in seconds (default: 5 seconds)")
	flag.IntVar(&cfg.DiscoveryInterval, "discovery_interval", cfg.DiscoveryInterval, "Discovery interval in seconds")
	flag.StringVar(&cfg.ZabbixServer, "zabbix_server", cfg.ZabbixServer, "The hostname of the zabbix server")
	flag.StringVar(&cfg.ZabbixHost, "zabbix_host", cfg.ZabbixHost, "The zabbix host to report data for")
	flag.BoolVar(&cfg.ZabbixSendDisabled, "disable_zabbix", false, "Disable zabbix sender")
	flag.BoolVar(&showStats, "show_stats_debug", false, "Show stats for debugging purposes")
	flag.BoolVar(&dumpStats, "dump_stats", false, "Dump stats")
	goflag.Set("logtostderr", "true")

	flag.Parse()
	flag.CommandLine.SortFlags = false

	cfg.LoadFile(configFile)

	glog.Infof("Starting apache_logpipe: output_logfile: %s, sending_interval: %d, discovery_interval: %d, zabbix_server: %s, zabbix_host: %s\n",
		cfg.OutputLogfile, cfg.SendingInterval, cfg.DiscoveryInterval, cfg.ZabbixServer, cfg.ZabbixHost)

	// Install signal handler
	signal.Notify(processing.SignalChan, syscall.SIGINT, syscall.SIGTERM)

	logSink := *processing.NewLogSink(cfg.OutputLogfile, cfg.OutputLogfileSymlink)

	requestAccounting := processing.NewRequestAccounting(*cfg)
	requestAccounting.DisableZabbixSender(cfg.ZabbixSendDisabled)

	if cfg.WebInterfaceEnable == true {
		wi := processing.NewWebInterface(*cfg)
		go wi.ServeRequests()
	}

	parseInput(logSink, *requestAccounting, *cfg)
	if showStats {
		requestAccounting.ShowStats()
	}
	if dumpStats {
		requestAccounting.DumpAccountingData()
	}
}
