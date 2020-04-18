
test:
	go test -v ./...
	go run apache_logpipe.go -v=1 -disable_zabbix -stderrthreshold=INFO --< testdata/test_access_log
	#timeout 10 go run apache_logpipe.go -v=1 -stderrthreshold=INFO -discovery_interval 3 -sending_interval 2 

exe:
	go get -d ./...
	go build apache_logpipe.go

release: format exe
	go test -coverprofile=cover.out -v ./... && go tool cover -html=cover.out

format:
	go fmt

