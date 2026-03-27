all:
	CGO_ENABLED=1 go run . real.db

test:
	CGO_ENABLED=1 go run . waow.db

build:
	CGO_ENABLED=1 go build .

run: build
	./archive-triage real.db