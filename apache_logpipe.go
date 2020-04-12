package main

// https://godoc.org/github.com/golang/glog

import (
	"apache_logpipe/processing"
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/pborman/getopt/v2"
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

func parseInput(verbose bool) {

	scanner := bufio.NewScanner(os.Stdin)

	lines := 0
	linesNotMatched := 0
	timeStart := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		lines++
		processing.WriteLogLine(line)
		match := lineRe.FindStringSubmatch(line)
		if len(match) == 0 {
			if verbose == true {
				log.Printf("not matched line: %s\n", line)
			}
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
			log.Fatalf("unable to convert code '%s' to a string", result["code"])
		}
		if code >= 400 || code < 200 {
			continue
		}
		matchStatic := requestStaticRe.FindStringSubmatch(result["uri"])
		if len(matchStatic) == 0 {
			processing.AccountRequest(result["domain"], result["uri"], result["time"], code)
		} else {
			processing.AccountRequest(result["domain"], "NOT MATCHED", result["time"], code)
		}
	}

	elapsed := time.Since(timeStart)
	linesPerSecond := float64(lines) / (float64(elapsed) / 1000000000)
	percentageNotMatched := (float64(linesNotMatched) / float64(lines)) * 100
	log.Printf("Processed %d lines in %s, %f lines per second, %d lines not matched (%0.2f%%)\n", lines, elapsed, linesPerSecond, linesNotMatched, percentageNotMatched)

	if verbose == true {
		processing.Showstats()
	}
}

func main() {

	helpFlag := getopt.BoolLong("help", '?', "display help")
	verboseFlag := getopt.BoolLong("verbose", 'v', "show debug informations")
	outputLogfile := getopt.StringLong("output_logfile", 'l', "/dev/null", "Filename with timestamp, i.e. '/var/log/apache2/access.log.%Y-%m-%d'")
	sendingInterval := getopt.IntLong("sending_interval", 'i', 300, "Sending interval in seconds")
	discoveryInterval := getopt.IntLong("discovery_interval", 'd', 900, "Discovery interval in seconds")
	zabbixServer := getopt.StringLong("zabbix_server", 'z', "127.0.0.1", "The zabbix server")
	zabbixHost := getopt.StringLong("zabbix_host", 'h', "127.0.0.1", "The zabbix host")
	getopt.Parse()
	//args := getopt.Args()
	//log.Printf("%t\n",*help_flag)
	if *helpFlag == true {
		getopt.Usage()
		os.Exit(1)
	}
	log.Printf("Starting apache_logpipe: output_logfile: %s, sending_interval: %d, discovery_interval: %d, zabbix_server: %s, zabbix_host: %s\n",
		*outputLogfile, *sendingInterval, *discoveryInterval, *zabbixServer, *zabbixHost)
	processing.FilenamePattern = *outputLogfile
	parseInput(*verboseFlag)
}
