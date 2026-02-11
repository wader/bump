# bump: golang /FROM golang:([\d.]+)/ docker:golang|^1
# bump: golang link "Release notes" https://golang.org/doc/devel/release.html
FROM golang:1.26.0-bookworm AS builder

# patch is used by cmd/bump/main_test.sh to test diff
RUN apt update && apt install -y patch

ARG GO111MODULE=on
WORKDIR $GOPATH/src/bump
COPY go.mod go.sum ./
RUN go mod download
COPY internal internal
COPY cmd cmd
RUN go test -v -cover -race ./...
RUN CGO_ENABLED=0 go build -o /bump -tags netgo -ldflags '-extldflags "-static"' ./cmd/bump
RUN cmd/bump/main_test.sh /bump

# bump: alpine /FROM alpine:([\d.]+)/ docker:alpine|^3
# bump: alpine link "Release notes" https://alpinelinux.org/posts/Alpine-$LATEST-released.html
FROM alpine:3.23.3 AS bump-base
# git is used by github action code
# curl for convenience
RUN apk add --no-cache \
    git \
    curl
COPY --from=builder /bump /usr/local/bin
RUN ["/usr/local/bin/bump", "version"]
RUN ["/usr/local/bin/bump", "pipeline", "git:https://github.com/torvalds/linux.git|*"]
ENTRYPOINT ["/usr/local/bin/bump"]

FROM bump-base AS bump-go
RUN apk add --no-cache go

FROM bump-base
