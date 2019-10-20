fetch:
	go get -d -v

build:
	CGO_ENABLED=0 go build -o ./bin/main main.go types.go iota-api.go

PHONY: run
run: build
	./bin/main

PHONY: deploy
deploy: push-heroku release-heroku

docker-build:
	docker build -t yodascholtz/kawapi:latest .

docker-run:
	docker run -p 8080:8080 yodascholtz/kawapi:latest

docker-push:
	docker push yodascholtz/kawapi:latest

push-heroku:
	heroku container:login && heroku container:push web

release-heroku:
	heroku container:login && heroku container:release web

logs:
	heroku logs --tail

format:
	go fmt