all:
	CGO_ENABLED=1 go run . real.db

test:
	CGO_ENABLED=1 go run . demo.db

build:
	CGO_ENABLED=1 go build .

run: build
	./archive-triage real.db

vhs: build
	rm -f demo.db || true
	docker run --rm -v ${PWD}:/vhs ghcr.io/charmbracelet/vhs demo/demo.tape