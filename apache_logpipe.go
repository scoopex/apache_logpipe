package main

//import "fmt"
import "log"
import "os"
import "time"
import "bufio"
import "regexp"
import "strconv"

// https://godoc.org/github.com/golang/glog
import "github.com/pborman/getopt/v2"

import "apache_logpipe/processing"

var help_flag bool
var verbose_flag bool
var output_logfile string
var sending_interval int
var discovery_interval int
var zabbix_server string
var zabbix_host string

var line_re = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`)

// regex insensitive static file ending
var request_static_re = regexp.MustCompile(`(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`)

func parse_input(verbose bool) {

	scanner := bufio.NewScanner(os.Stdin)

	lines := 0
	lines_not_matched := 0
	time_start := time.Now()
	for scanner.Scan() {
		line := scanner.Text()
		lines++
		match := line_re.FindStringSubmatch(line)
		if len(match) == 0 {
			if verbose == true {
				log.Printf("not matched line: %s\n", line)
			}
			lines_not_matched++
			continue
		}
		result := make(map[string]string)
		for i, name := range line_re.SubexpNames() {
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
		match_static := request_static_re.FindStringSubmatch(result["uri"])
		if len(match_static) == 0 {
			processing.Account_request(result["domain"], result["uri"], result["time"], code)
		} else {
			processing.Account_request(result["domain"], "NOT MATCHED", result["time"], code)
		}
	}

	elapsed := time.Since(time_start)
	lines_per_second := float64(lines) / (float64(elapsed) / 1000000000)
	percentage_not_matched := (float64(lines_not_matched) / float64(lines)) * 100
	log.Printf("Processed %d lines in %s, %f lines per second, %d lines not matched (%0.2f%%)\n", lines, elapsed, lines_per_second, lines_not_matched, percentage_not_matched)

	if verbose == true {
		processing.Showstats()
	}
}

func main(){

	help_flag := getopt.BoolLong("help", '?', "display help")
	verbose_flag := getopt.BoolLong("verbose", 'v', "show debug informations")
	output_logfile := getopt.StringLong("output_logfile", 'l', "/dev/null", "Filename with timestamp, i.e. '/var/log/apache2/access.log.%%Y-%%m-%%d'")
	sending_interval := getopt.IntLong("sending_interval", 'i', 300, "Sending interval in seconds")
	discovery_interval := getopt.IntLong("discovery_interval", 'd', 900, "Discovery interval in seconds")
	zabbix_server := getopt.StringLong("zabbix_server", 'z', "127.0.0.1", "The zabbix server")
	zabbix_host := getopt.StringLong("zabbix_host", 'h', "127.0.0.1", "The zabbix host")
	getopt.Parse()
	//args := getopt.Args()
	//log.Printf("%t\n",*help_flag)
	if *help_flag == true {
		getopt.Usage()
		os.Exit(1)
	}
	log.Printf("Starting apache_logpipe: output_logfile: %s, sending_interval: %d, discovery_interval: %d, zabbix_server: %s, zabbix_host: %s\n",
		*output_logfile, *sending_interval, *discovery_interval, *zabbix_server, *zabbix_host)
	parse_input(*verbose_flag)
}
