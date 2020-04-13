
test:
	go test -v ./...
	go run apache_logpipe.go -v=1 -stderrthreshold=INFO --< testdata/test_access_log

exe:
	go get -d ./...
	go build apache_logpipe.go

release: format exe
	go test -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

format:
	go fmt

