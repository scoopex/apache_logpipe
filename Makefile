run:
	go run apache_logpipe.go -v=1 -disable_zabbix -stderrthreshold=INFO --< testdata/test_access_log
	#timeout 10 go run apache_logpipe.go -v=1 -stderrthreshold=INFO -discovery_interval 3 -sending_interval 2 

test:
	go get -d ./...
	go test -coverprofile=cover.out -count=1  -v ./... 

exe:
	go build apache_logpipe.go

clean:
	rm -f apache_logpipe cover.out

release: clean format test exe
	go tool cover -html=cover.out

format:
	go fmt

