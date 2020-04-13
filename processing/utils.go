package processing

import (
	"os"
	"runtime"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// Debugit displays the give datastructure
func Debugit(exit bool, debug ...interface{}) {
	scs := spew.ConfigState{
		SortKeys: true,
		Indent:   " ",
	}
	pc, fn, line, _ := runtime.Caller(1)
	glog.Infof("Debug output %s[%s:%d] \n>>>\n%s<<<", runtime.FuncForPC(pc).Name(), fn, line, scs.Sdump(debug...))
	if exit {
		os.Exit(1)
	}
}
