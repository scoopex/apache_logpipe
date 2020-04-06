
https://golang.org/dl/
https://www.alexedwards.net/blog/an-overview-of-go-tooling

sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt update
sudo apt install golang-go

go get github.com/pborman/getopt
go get -d ./...
#go get -u -v -f all
go run apache_logpipe.go <  test_log
go run apache_logpipe.go --verbose < test_log_short
go build apache_logpipe.go
