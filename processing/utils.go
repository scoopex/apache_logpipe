package processing

import (
	"flag"
	"os"
	"runtime"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// Debugit displays the give datastructure
func Debugit(exit bool, note string, debug ...interface{}) {
	scs := spew.ConfigState{
		SortKeys: true,
		Indent:   " ",
	}
	pc, fn, line, _ := runtime.Caller(1)
	glog.Infof("%s - Debug output %s[%s:%d] \n>>>\n%s<<<", note, runtime.FuncForPC(pc).Name(), fn, line, scs.Sdump(debug...))
	if exit {
		os.Exit(1)
	}
}

func GetHostname() string {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return name
}

// SetupGlogForTests perform initalization of test logging
func SetupGlogForTests() {
	flag.Set("logtostderr", "true")
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "4", "test")
	flag.Lookup("v").Value.Set(logLevel)
}

// FileExists checks fopr file if it exists
func FileExists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
