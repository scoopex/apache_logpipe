package processing

import (
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
)

// Debugit displays the give datastructure
func Debugit(exit bool, debug ...interface{}) {
	scs := spew.ConfigState{
		SortKeys: true,
		Indent:   " ",
	}
	log.Println(scs.Sdump(debug...))
	if exit {
		os.Exit(1)
	}
}
