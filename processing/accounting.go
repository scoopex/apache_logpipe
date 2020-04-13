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
type Accounting struct {
	classes         []int
	requestMappings map[string]*regexp.Regexp
	stats           map[string]map[string]*AccountingSet
}

// Config configures the accounting
var RequestAccounting = Accounting{
	// a list of accounting classes, defined in microseconds
	classes: []int{0, 500000, 10000000, 5000000, 60000000, 300000000},
	// a map of requesttypes containing compiled regexes
	requestMappings: map[string]*regexp.Regexp{
		"all": regexp.MustCompile(`([^?]*)\??.*`),
	},
	// the current state of the statistics
	stats: map[string]map[string]*AccountingSet{},
}

func (c *Accounting) getPerfclasses(responsetime int) int {
	for _, perfclass := range c.classes {
		if responsetime >= perfclass {
			return perfclass
		}
	}
	return 0
}

// AccountRequest accounts the request :-)
func (c *Accounting) AccountRequest(domain string, uri string, time string, code int) {
	responsetime, err := strconv.Atoi(time)
	if err != nil {
		log.Fatalf("unable to convert time '%s' to a string", time)
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
				codes:   make(map[int]int),
				classes: make(map[int]int),
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
	Debugit(false, c.stats)
}
