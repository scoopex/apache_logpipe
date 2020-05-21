run:
	testdata/create_testdata
	go run apache_logpipe.go -v=1 -disable_zabbix -stderrthreshold=INFO --< testdata/test_access_log
	go run apache_logpipe.go -disable_zabbix --< testdata/test_access_log_huge

test:
	go get -d ./...
	go test -coverprofile=cover.out -timeout 10s -count=1 -v ./... 

exe:
	go build apache_logpipe.go

clean:
	rm -f apache_logpipe cover.out rm testdata/test_access_log_huge

release: clean format test exe
	go tool cover -html=cover.out

format:
	go fmt

