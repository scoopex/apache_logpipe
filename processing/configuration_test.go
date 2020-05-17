package processing_test

import (
	"apache_logpipe/processing"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

var flagSetOrig *flag.FlagSet

func init() {
	SetupGlogForTests()
}

func TestConfigurationSimple(t *testing.T) {
	cfg := processing.NewConfiguration()
	cfg.ZabbixServer = "zabbix3.foo.bar"
	assert.Equal(t, "zabbix3.foo.bar", cfg.ZabbixServer, "Commandline flag has higher precedence than default value")
}

func TestConfigurationFile(t *testing.T) {
	exampleFile := GetProjectBaseDir() + "/config.example.ini"
	cfg := processing.NewConfiguration()
	cfg.DiscoveryInterval = 22
	cfg.LoadFile(exampleFile)
	assert.Equal(t, "zabbix.host.edu", cfg.ZabbixServer, "Config file has higher precedence than default value")
	assert.Equal(t, 22, cfg.DiscoveryInterval, "Commandline flag has higher precedence than config file value")
	assert.IsType(t, cfg.ResponstimeClasses, []int{})
	assert.True(t, len(cfg.RequestMappings) == 2)
}
