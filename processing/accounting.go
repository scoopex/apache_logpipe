package processing

import (
	"log"
	"regexp"
	"strconv"
)

// AccountingSet for a certain request type
type AccountingSet struct {
	count   int64
	sum     int64
	codes   map[int]int
	classes map[int]int
}

// map domain -> group -> AccountingStruct
var stats = map[string]map[string]*AccountingSet{}

var classes = []int{0, 500000, 10000000, 5000000, 60000000, 300000000}

var requestMappings = map[string]*regexp.Regexp{
	"all": regexp.MustCompile(`([^?]*)\??.*`),
}

func getPerfclasses(responsetime int) int {
	for _, perfclass := range classes {
		if responsetime >= perfclass {
			return perfclass
		}
	}
	return 0
}

// AccountRequest accounts the request :-)
func AccountRequest(domain string, uri string, time string, code int) {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		log.Fatalf("unable to convert time '%s' to a string", time)
	}

	for name, reName := range requestMappings {
		match := reName.FindStringSubmatch(uri)
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
		stats[domain][name].classes[getPerfclasses(responsetime)]++
	}
}

// Showstats displays the statistics
func Showstats() {
	Debugit(stats)
}
