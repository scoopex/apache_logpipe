package processing

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"gopkg.in/ini.v1"
)

type Configuration struct {
	OutputLogfile            string
	OutputLogfileSymlink     string
	SendingInterval          int
	Timeout                  int
	DiscoveryInterval        int
	ZabbixServer             string
	ZabbixHost               string
	ZabbixSendDisabled       bool
	ResponstimeClasses       []int
	RequestMappings          map[string]*regexp.Regexp
	configFile               string
	RegexLogLineString       string
	RegexStaticContentString string
	FractionOfSecond         int
	WebInterfaceListen       string
	WebInterfaceEnable       bool
	WebInterfaceUser         string
	WebInterfacePassword     string
}

// NewConfiguration create a new Configuration object
func NewConfiguration() *Configuration {
	var cfg = new(Configuration)
	cfg.configFile = ""
	cfg.OutputLogfile = "/dev/null"
	cfg.OutputLogfileSymlink = ""
	cfg.SendingInterval = 120
	cfg.Timeout = 900
	cfg.ZabbixServer = "zabbix"
	cfg.ZabbixHost = GetHostname()
	cfg.ZabbixSendDisabled = false
	cfg.RegexLogLineString = `^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`
	cfg.RegexStaticContentString = `(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`
	cfg.ResponstimeClasses = []int{0, 500000, 10000000, 5000000, 60000000, 300000000}
	cfg.RequestMappings = map[string]*regexp.Regexp{
		"all": regexp.MustCompile(`([^?]*)\??.*`),
	}
	cfg.FractionOfSecond = 1000000000
	cfg.WebInterfaceListen = "127.0.0.1:10080"
	cfg.WebInterfaceUser = "admin"
	cfg.WebInterfacePassword = "admin"
	cfg.WebInterfaceEnable = true
	return cfg
}

// LoadFile loads the values defined in the file
func (c *Configuration) LoadFile(configFile string) {

	c.configFile = configFile

	if c.configFile == "" {
		glog.Info("No config file specified, using defaults and commandline args only")
		return
	}

	// https://ini.unknwon.io/docs/intro/getting_started
	var iniFile *ini.File = nil
	var err error = nil

	glog.Infof("Loading config file >>>%s<<< now", c.configFile)
	iniFile, err = ini.Load(c.configFile)
	if err != nil {
		glog.Errorf("Failed to read file: %v", err)
		os.Exit(1)
	}

	defaultCfg := NewConfiguration()

	c.OutputLogfile = getStringValue(iniFile, "global", "output_logile", c.OutputLogfile, defaultCfg.OutputLogfile)
	c.OutputLogfileSymlink = getStringValue(iniFile, "global", "symlink", c.OutputLogfileSymlink, defaultCfg.OutputLogfileSymlink)
	c.SendingInterval = getIntValue(iniFile, "global", "sending_interval", c.SendingInterval, defaultCfg.SendingInterval)
	c.Timeout = getIntValue(iniFile, "global", "timeout", c.Timeout, defaultCfg.Timeout)
	c.DiscoveryInterval = getIntValue(iniFile, "global", "discovery_interval", c.DiscoveryInterval, defaultCfg.DiscoveryInterval)
	c.ZabbixServer = getStringValue(iniFile, "global", "zabbix_server", c.ZabbixServer, defaultCfg.ZabbixServer)
	c.ZabbixHost = getStringValue(iniFile, "global", "zabbix_host", c.ZabbixHost, defaultCfg.ZabbixHost)
	c.FractionOfSecond = getIntValue(iniFile, "global", "fraction_of_second", c.FractionOfSecond, defaultCfg.FractionOfSecond)

	c.RegexLogLineString = getStringValue(iniFile, "global", "regex_logline", "", defaultCfg.RegexLogLineString)
	c.RegexStaticContentString = getStringValue(iniFile, "global", "regex_static_content", "", defaultCfg.RegexStaticContentString)
	c.ResponstimeClasses = getResponseTimeClasses(iniFile, "global", "request_mappings", defaultCfg.ResponstimeClasses)
	c.RequestMappings = getRequestMappings(iniFile, defaultCfg.RequestMappings)

}

func getRequestMappings(iniFile *ini.File, defaultValue map[string]*regexp.Regexp) map[string]*regexp.Regexp {
	if iniFile == nil {
		return defaultValue
	}
	newRequestMappings := map[string]*regexp.Regexp{}
	for _, section := range iniFile.SectionStrings() {
		if section == "global" || section == "DEFAULT" {
			continue
		}
		if iniFile.Section(section).HasKey("regex") {
			glog.V(1).Infof("parsed request mappings from file: name: >>>%s<<<, regex >>>%s<<<", section, iniFile.Section(section).Key("regex").String())
			newRequestMappings[section] = regexp.MustCompile(iniFile.Section(section).Key("regex").String())
		}
	}
	if len(newRequestMappings) > 0 {
		return newRequestMappings
	}
	return defaultValue
}

func getResponseTimeClasses(iniFile *ini.File, section string, key string, defaultValue []int) []int {
	if iniFile != nil && iniFile.Section(section).HasKey(key) {
		classesByString := strings.Split(iniFile.Section(section).Key(key).String(), ",")
		classesByInteger := make([]int, 10)
		for _, classStr := range classesByString {
			classInt, err := strconv.Atoi(strings.TrimSpace(classStr))
			if err != nil {
				glog.Fatalf("unable to convert perf class  '%s' to a integer", classStr)
			}
			classesByInteger = append(classesByInteger, classInt)
		}
		return classesByInteger
	}
	return defaultValue
}

func getStringValue(iniFile *ini.File, section string, key string, currentValue string, defaultValue string) string {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == defaultValue {
		return iniFile.Section(section).Key(key).MustString(defaultValue)
	}
	if currentValue == "" {
		return defaultValue
	}
	return currentValue

}

func getIntValue(iniFile *ini.File, section string, key string, currentValue int, defaultValue int) int {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == defaultValue {
		return iniFile.Section(section).Key(key).MustInt(defaultValue)
	}
	if currentValue == 0 {
		return defaultValue
	}
	return currentValue

}
