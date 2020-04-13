https://medium.com/rate-engineering/go-test-your-code-an-introduction-to-effective-testing-in-go-6e4f66f2c259
https://golang.org/dl/
https://www.alexedwards.net/blog/an-overview-of-go-tooling

go get github.com/pborman/getopt
go get -d ./...
#go get -u -v -f all
go run apache_logpipe.go <  test_log
go run apache_logpipe.go --verbose < test_log_short
go build apache_logpipe.go
