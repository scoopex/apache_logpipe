package processing

import "log"
import "os"

// https://godoc.org/github.com/golang/glog
import "github.com/davecgh/go-spew/spew"


func Debugit(debug ...interface{}) {
   scs := spew.ConfigState{
      SortKeys: true,
      Indent:   " ",
   }
   log.Println(scs.Sdump(debug))
   os.Exit(1)
}

func Showstats(){
   Debugit(stats)
}
