fetch:
	go get -d -v

build:
	CGO_ENABLED=0 go build -o ./bin/main main.go types.go iota-api.go

docker-build:
	docker build -t kawapi:latest .

docker-run:
	docker run -p 8080:8080 kawapi:latest
