package processing

import (
	"os"
	"regexp"

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
	RegexLogline             regexp.Regexp
	RegexStaticContent       regexp.Regexp
	configFile               string
	regexLogLineString       string
	regexStaticContentString string
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
	cfg.regexLogLineString = `^\d+\.\d+\.\d+\.\d+ (?P<domain>[^ ]+?)\s.*] "(GET|POST|PUT|PROPFIND|OPTIONS|DELETE) (?P<uri>/[^ ]*?)(?P<getparam>\?[^ ]*?)? HTTP.*" (?P<code>\d+) .* (?P<time>\d+)$`
	cfg.regexStaticContentString = `(?i).+\.(gif|jpg|jpeg|png|ico|flv|swf|js|css|txt|woff|ttf)`
	cfg.RegexLogline = *regexp.MustCompile(cfg.regexStaticContentString)
	cfg.RegexStaticContent = *regexp.MustCompile(cfg.regexStaticContentString)
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

	c.RegexLogline = getRegexValue(iniFile, "global", "regex_logline", "", defaultCfg.regexLogLineString)
	c.RegexStaticContent = getRegexValue(iniFile, "global", "regex_static_content", "", defaultCfg.regexStaticContentString)

	for _, section := range iniFile.SectionStrings() {
		if section == "global" || section == "DEFAULT" {
			continue
		}
		glog.Infof(">>>>>>>>>>>>>>>>>>>>> %s<<<<<<<<<<<<<<<<<<", section)
	}
}

func getRegexValue(iniFile *ini.File, section string, key string, currentValue string, defaultValue string) regexp.Regexp {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == defaultValue {
		return *regexp.MustCompile(iniFile.Section(section).Key(key).MustString(defaultValue))
	}
	if currentValue == "" {
		return *regexp.MustCompile(defaultValue)
	}
	return *regexp.MustCompile(defaultValue)

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
