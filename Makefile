
test:
	go test -v ./...
	go run apache_logpipe.go --verbose --< testdata/test_access_log

exe:
	go get github.com/lestrrat-go/strftime
	go get github.com/pborman/getopt
	go get github.com/davecgh/go-spew/spew
	go build apache_logpipe.go

release: exe
	go test -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

format:
	go fmt

