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
	cp -a examples/* "${TEMPDIR}"
	go build -o "${TEMPDIR}/bump" cmd/bump/main.go
	go build -o "${TEMPDIR}/filtersmarkdown" _dev/filtersmarkdown.go
	cd "${TEMPDIR}" ; \
		cat "${REPODIR}/README.md" | PATH="${TEMPDIR}:${PATH}" go run "${REPODIR}/_dev/mdsh.go" > "${TEMPDIR}/README.md"
	mv "${TEMPDIR}/README.md" "${REPODIR}/README.md"
	rm -rf "${TEMPDIR}"

actions:
	for i in $(shell cd action && ls -1 | grep -v .yml) ; do \
		cat action/action.yml | sed -E "s/image: .*/image: 'docker:\/\/mwader\/bump:$$i'/" > action/$$i/action.yml ; \
	done

.PHONY: README.md test cover lint
