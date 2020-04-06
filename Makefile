
test:
	go run apache_logpipe.go --verbose < testdata/test_access_log

test_long:
	go run apache_logpipe.go --verbose < test_log



exe:
	go get github.com/pborman/getopt
	go get github.com/davecgh/go-spew/spew
	go build apache_logpipe.go
format:
	go fmt

