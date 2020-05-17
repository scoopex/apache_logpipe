package processing_test

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/golang/glog"
)

var glogReady bool = false
var mutex sync.Mutex

func SetupLogfileTestDir() string {
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if !ok || details == nil {
		glog.Error("unable to identify caller")
		os.Exit(1)
	}

	dest := fmt.Sprintf("/tmp/%s_%d", details.Name(), rand.Intn(1000000))

	err := os.MkdirAll(dest, os.FileMode(0777))
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to create destination dir '%s' - %s\n", dest, err.Error())
	}
	glog.Infof("Dest Temporary Dir: %s", dest)
	return dest
}

func RemoveTestDir(testDir string) {
	glog.Infof("Removing testdir : %s", testDir)
	err := os.RemoveAll(testDir)
	if err != nil {
		glog.Errorf("unable to remove test dir %s : %s", testDir, err.Error())
	}
}

// SetupGlogForTests perform initalization of test logging
func SetupGlogForTests() {
	mutex.Lock()
	defer mutex.Unlock()
	if glogReady {
		return
	}
	// flag.Parse()
	flag.Set("logtostderr", "true")
	var logLevel string
	flag.StringVar(&logLevel, "logLevel", "2", "test")
	flag.Lookup("v").Value.Set(logLevel)
	glogReady = true
}

func GetProjectBaseDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	return dir
}
