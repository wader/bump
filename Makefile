all: README.md test

test:
	go test -v -cover -race ./...

cover:
	go test -cover -race -coverpkg=./... -coverprofile=cover.out ./...
	go tool cover -func=cover.out

lint:
	golangci-lint run

README.md:
	$(eval REPODIR=$(shell pwd))
	$(eval TEMPDIR=$(shell mktemp -d))
	cp -a examples "${TEMPDIR}"
	go build -o "${TEMPDIR}/bump" cmd/bump/main.go
	go build -o "${TEMPDIR}/filtersmarkdown" _dev/filtersmarkdown.go
	cd "${TEMPDIR}" ; \
		cat "${REPODIR}/_dev/README.md" | PATH="${TEMPDIR}:${PATH}" go run "${REPODIR}/_dev/mdsh.go" > "${REPODIR}/README.md"
	rm -rf "${TEMPDIR}"

.PHONY: README.md test cover lint
