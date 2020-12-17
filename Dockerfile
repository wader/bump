# bump: golang /FROM golang:([\d.]+)/ docker:golang|^1
FROM golang:1.15.6-buster AS builder

# patch is used by naivediff package tests (compares diff output)
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
FROM alpine:3.12.3
# git is used by github action code
RUN apk add --no-cache git
COPY --from=builder /bump /usr/local/bin
RUN ["/usr/local/bin/bump", "version"]
RUN ["/usr/local/bin/bump", "pipeline", "git:https://github.com/torvalds/linux.git|*"]
ENTRYPOINT ["/usr/local/bin/bump"]
