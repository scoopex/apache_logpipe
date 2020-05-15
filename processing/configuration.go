package processing

import (
	"flag"
	"os"
	"regexp"

	"github.com/golang/glog"
	"gopkg.in/ini.v1"
)

type Configuration struct {
	OutputLogfile            *string
	OutputLogfileSymlink     *string
	SendingInterval          *int
	Timeout                  *int
	DiscoveryInterval        *int
	ZabbixServer             *string
	ZabbixHost               *string
	ZabbixSendDisabled       *bool
	regexLogLineString       *string
	regexStaticContentString *string
	RegexStaticContent       *regexp.Regexp
	RegexLogline             *regexp.Regexp
	configFile               *string
}

// NewConfiguration create a new Configuration object
func NewConfiguration() *Configuration {

	var cfg = new(Configuration)

	/*
	* Parsing the arguments
	 */
	cfg.configFile = flag.String("config", "", "Name of the config file")
	cfg.OutputLogfile = flag.String("output_logfile", "", "Filename with timestamp, i.e. '/var/log/apache2/access.log.%Y-%m-%d'")
	cfg.OutputLogfileSymlink = flag.String("symlink", "", "A symlink which points to the current logfile")
	cfg.SendingInterval = flag.Int("sending_interval", 0, "Sending interval in seconds (default: 120 seconds)")
	cfg.Timeout = flag.Int("timeout", 0, "timeout in seconds (default: 5 seconds)")
	cfg.DiscoveryInterval = flag.Int("discovery_interval", 0, "Discovery interval in seconds (default: 900 seconds)")
	cfg.ZabbixServer = flag.String("zabbix_server", "", "The zabbix server (default: 'zabbix')")
	cfg.ZabbixHost = flag.String("zabbix_host", "", "The zabbix host (default: "+GetHostname()+")")
	cfg.ZabbixSendDisabled = flag.Bool("disable_zabbix", false, "Disable zabbix sender")

	flag.Set("logtostderr", "true")
	flag.Parse()
	return cfg
}

func (c *Configuration) LoadFile() {

	// https://ini.unknwon.io/docs/intro/getting_started
	var iniFile *ini.File = nil
	var err error = nil
	if *c.configFile != "" {
		glog.Infof("Loading config file >>>%s<<< now", *c.configFile)
		iniFile, err = ini.Load(*c.configFile)
		if err != nil {
			glog.Errorf("Failed to read file: %v", err)
			os.Exit(1)
		}
	}
	*c.OutputLogfile = getStringValue(iniFile, "global", "output_logile", *c.OutputLogfile, "/dev/null")
	*c.OutputLogfileSymlink = getStringValue(iniFile, "global", "symlink", *c.OutputLogfileSymlink, "")
	*c.SendingInterval = getIntValue(iniFile, "global", "sending_interval", *c.SendingInterval, 120)
	*c.Timeout = getIntValue(iniFile, "global", "timeout", *c.Timeout, 5)
	*c.DiscoveryInterval = getIntValue(iniFile, "global", "discovery_interval", *c.DiscoveryInterval, 900)
	*c.ZabbixServer = getStringValue(iniFile, "global", "zabbix_server", *c.ZabbixServer, "zabbix")
	*c.ZabbixHost = getStringValue(iniFile, "global", "zabbix_host", *c.ZabbixHost, GetHostname())

	regexLogLineString := `^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`
	regexStaticContentString := `(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`
	c.RegexLogline = getRegexValue(iniFile, "global", "regex_logline", regexLogLineString, regexLogLineString)
	c.RegexStaticContent = getRegexValue(iniFile, "global", "regex_static_content", regexStaticContentString, regexStaticContentString)
}

func getRegexValue(iniFile *ini.File, section string, key string, currentValue string, defaultValue string) *regexp.Regexp {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == "" {
		return regexp.MustCompile(iniFile.Section(section).Key(key).MustString(defaultValue))
	}
	if currentValue == "" {
		return regexp.MustCompile(defaultValue)
	}
	return regexp.MustCompile(defaultValue)

}

func getStringValue(iniFile *ini.File, section string, key string, currentValue string, defaultValue string) string {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == "" {
		return iniFile.Section(section).Key(key).MustString(defaultValue)
	}
	if currentValue == "" {
		return defaultValue
	}
	return currentValue

}

func getIntValue(iniFile *ini.File, section string, key string, currentValue int, defaultValue int) int {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == 0 {
		return iniFile.Section(section).Key(key).MustInt(defaultValue)
	}
	if currentValue == 0 {
		return defaultValue
	}
	return currentValue

}
