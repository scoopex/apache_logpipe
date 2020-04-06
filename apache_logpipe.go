package main

//import "fmt"
import "log"
import "os"
import "time"
import "bufio"
import "regexp"
import "strconv"

// https://godoc.org/github.com/golang/glog
import "github.com/davecgh/go-spew/spew"
import "github.com/pborman/getopt/v2"

var help_flag bool
var verbose_flag bool
var output_logfile string
var sending_interval int
var discovery_interval int
var zabbix_server string
var zabbix_host string

type AccountingSet struct {
	count   int64
	sum     int64
	codes   map[int]int
	classes map[int]int
}

// map domain -> group -> AccountingStruct
var stats = map[string]map[string]*AccountingSet{}

var classes = []int{0, 500000, 10000000, 5000000, 60000000, 300000000}

var line_re = regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`)

// regex insensitive static file ending
var request_static_re = regexp.MustCompile(`(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`)

var request_mappings = map[string]*regexp.Regexp{
	"all": regexp.MustCompile(`([^?]*)\??.*`),
}

func debugit(debug ...interface{}) {
	scs := spew.ConfigState{
		SortKeys: true,
		Indent:   " ",
	}
	log.Println(scs.Sdump(debug))
	os.Exit(1)
}

func get_perfclass(responsetime int) int {
	for _, perfclass := range classes {
		if responsetime >= perfclass {
			return perfclass
		}
	}
	return 0
}

func account_request(domain string, uri string, time string, code int) {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		log.Fatalf("unable to convert time '%s' to a string", time)
	}

	for name, re_name := range request_mappings {
		match := re_name.FindStringSubmatch(uri)
		if len(match) == 0 {
			continue
		}
		if (stats[domain] == nil) || (stats[domain][name] == nil) {
			stats[domain] = make(map[string]*AccountingSet)
			stats[domain][name] = &AccountingSet{
				count:   0,
				sum:     0,
				codes:   make(map[int]int),
				classes: make(map[int]int),
			}
			for _, perfclass := range classes {
				stats[domain][name].classes[perfclass] = 0
			}
		}

		stats[domain][name].sum += int64(responsetime)
		stats[domain][name].count++
		stats[domain][name].codes[code]++
		stats[domain][name].classes[get_perfclass(responsetime)]++
	}
}

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
			account_request(result["domain"], result["uri"], result["time"], code)
		} else {
			account_request(result["domain"], "NOT MATCHED", result["time"], code)
		}
	}

	elapsed := time.Since(time_start)
	lines_per_second := float64(lines) / (float64(elapsed) / 1000000000)
	percentage_not_matched := (float64(lines_not_matched) / float64(lines)) * 100
	log.Printf("Processed %d lines in %s, %f lines per second, %d lines not matched (%0.2f%%)\n", lines, elapsed, lines_per_second, lines_not_matched, percentage_not_matched)

	if verbose == true {
		debugit(stats)
	}

}

func main() {

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
