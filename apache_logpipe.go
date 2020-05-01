package main

import (
	"apache_logpipe/processing"
	"bufio"
	"flag"
	"os"
	"os/signal"
	"regexp"
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

var lineRe = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`)

// regex insensitive static file ending
var requestStaticRe = regexp.MustCompile(`(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`)

func parseInput(logSink processing.LogSink, requestAccounting processing.RequestAccounting) {

	scanner := bufio.NewScanner(os.Stdin)

	var lines int64 = 0
	var linesNotMatched int64 = 0
	timeStart := time.Now()

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
			glog.Fatalf("unable to convert code '%s' to a string", result["code"])
		}
		if code >= 400 || code < 200 {
			linesNotMatched++
			continue
		}
		matchStatic := requestStaticRe.FindStringSubmatch(result["uri"])
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

	processing.PerfSetChan <- processing.PerfSet{
		Domain: "COMPLETE",
		Ident:  "COMPLETE",
		Time:   "0",
		Code:   1,
	}

	linesAccounted := <-processing.CompleteChan
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
	outputLogfile := flag.String("output_logfile", "/dev/null", "Filename with timestamp, i.e. '/var/log/apache2/access.log.%Y-%m-%d'")
	outputLogfileSymlink := flag.String("symlink", "", "A symlink which points to the current logfile")
	sendingInterval := flag.Int("sending_interval", 300, "Sending interval in seconds")
	timeout := flag.Int("timeout", 5, "timeout in seconds")
	discoveryInterval := flag.Int("discovery_interval", 900, "Discovery interval in seconds")
	zabbixServer := flag.String("zabbix_server", "127.0.0.1", "The zabbix server")
	zabbixHost := flag.String("zabbix_host", "127.0.0.1", "The zabbix host")
	zabbixSendDisabled := flag.Bool("disable_zabbix", false, "Disable zabbix sender")
	flag.Set("logtostderr", "true")
	flag.Parse()

	glog.Infof("Starting apache_logpipe: output_logfile: %s, sending_interval: %d, discovery_interval: %d, zabbix_server: %s, zabbix_host: %s\n",
		*outputLogfile, *sendingInterval, *discoveryInterval, *zabbixServer, *zabbixHost)

	// Install signal handler
	signal.Notify(processing.SignalChan, syscall.SIGINT, syscall.SIGTERM)

	logSink := *processing.NewLogSink(*outputLogfile, *outputLogfileSymlink)

	requestAccounting := processing.NewRequestAccounting(*discoveryInterval, *sendingInterval, *timeout)
	requestAccounting.DisableZabbixSender(*zabbixSendDisabled)
	parseInput(logSink, *requestAccounting)
}
