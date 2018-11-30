.PHONY: test

gofmt:
	./scripts/gofmt.sh

test:
	./scripts/run-tests.sh

build:
	env GOOS=linux GOARCH=amd64 go build -o bin/linux-amd64/hedwig-models-generator .
	env GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64/hedwig-models-generator .
	cd bin/linux-amd64 && zip hedwig-models-generator-linux-amd64.zip hedwig-models-generator; cd -
	cd bin/darwin-amd64 && zip hedwig-models-generator-darwin-amd64.zip hedwig-models-generator; cd -
