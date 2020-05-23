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
	data            RequestAccounting
}

// NewWebInterface return the instance
func NewWebInterface(cfg Configuration, data RequestAccounting) *WebInterface {
	// RequestAccountingInst configures the accounting
	WebInterfaceInst := WebInterface{
		ListenInterface: cfg.WebInterfaceListen,
		User:            cfg.WebInterfaceUser,
		Password:        cfg.WebInterfacePassword,
		data:            data,
	}
	return &WebInterfaceInst
}

func (c *WebInterface) getStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, c.data.GetJsonStats())
}

// ServeRequests start the serving of requests
func (c *WebInterface) ServeRequests() {

	glog.Infof("start serving request on %s", c.ListenInterface)

	http.Handle("/getStatus", httpauth.SimpleBasicAuth(c.User, c.Password)(http.HandlerFunc(c.getStatus)))

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.ListenAndServe(c.ListenInterface, nil)
}
