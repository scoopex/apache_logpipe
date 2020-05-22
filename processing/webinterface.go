package processing

import (
	"fmt"
	"net/http"

	"github.com/goji/httpauth"
	"github.com/golang/glog"
)

type WebInterface struct {
	ListenInterface string
	User            string
	Password        string
}

// NewWebInterface return the instance
func NewWebInterface(cfg Configuration) *WebInterface {
	// RequestAccountingInst configures the accounting
	WebInterfaceInst := WebInterface{
		ListenInterface: cfg.WebInterfaceListen,
		User:            cfg.WebInterfaceUser,
		Password:        cfg.WebInterfacePassword,
	}
	return &WebInterfaceInst
}

func showStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "some message to authenticated user.")
}

// ServeRequests start the serving of requests
func (c *WebInterface) ServeRequests() {
	glog.Infof("start serving request on %s", c.ListenInterface)

	http.Handle("/", httpauth.SimpleBasicAuth(c.User, c.Password)(http.HandlerFunc(showStatus)))

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(c.ListenInterface, nil)
}
