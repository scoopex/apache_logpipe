package processing

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"gopkg.in/ini.v1"
)

type Configuration struct {
	OutputLogfile        *string
	OutputLogfileSymlink *string
	SendingInterval      *int
	Timeout              *int
	DiscoveryInterval    *int
	ZabbixServer         *string
	ZabbixHost           *string
	ZabbixSendDisabled   *bool
}

// https://ini.unknwon.io/docs/intro/getting_started
// NewConfiguration create a new Configuration object
func NewConfiguration() *Configuration {

	var cfg = new(Configuration)

	/*
	* Parsing the arguments
	 */
	configFile := flag.String("config", "", "Name of the config file")
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

	var iniFile *ini.File = nil
	var err error = nil
	if *configFile != "" {
		iniFile, err = ini.Load(*configFile)
		if err != nil {
			glog.Errorf("Failed to read file: %v", err)
			os.Exit(1)
		}
	}
	*cfg.OutputLogfile = getStringValue(iniFile, "", "output_logile", *cfg.OutputLogfile, "/dev/null")
	*cfg.OutputLogfileSymlink = getStringValue(iniFile, "", "symlink", *cfg.OutputLogfileSymlink, "")
	*cfg.SendingInterval = getIntValue(iniFile, "", "sending_interval", *cfg.SendingInterval, 120)
	*cfg.Timeout = getIntValue(iniFile, "", "timeout", *cfg.Timeout, 5)
	*cfg.DiscoveryInterval = getIntValue(iniFile, "", "discovery_interval", *cfg.DiscoveryInterval, 900)
	*cfg.ZabbixServer = getStringValue(iniFile, "", "zabbix_server", *cfg.ZabbixServer, "zabbix")
	*cfg.ZabbixHost = getStringValue(iniFile, "", "zabbix_host", *cfg.ZabbixHost, GetHostname())

	return cfg
}

func getStringValue(iniFile *ini.File, section string, key string, currentValue string, defaultValue string) string {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == "" {
		return iniFile.Section("").Key(key).MustString(defaultValue)
	}
	if currentValue == "" {
		return defaultValue
	}
	return currentValue

}

func getIntValue(iniFile *ini.File, section string, key string, currentValue int, defaultValue int) int {
	if iniFile != nil && iniFile.Section(section).HasKey(key) && currentValue == 0 {
		return iniFile.Section("").Key(key).MustInt(defaultValue)
	}
	if currentValue == 0 {
		return defaultValue
	}
	return currentValue

}
