FROM golang:1.23.12

ENV GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN go build -mod=vendor -o /fnexecd ./cmd/fnexecd

EXPOSE 8080

CMD ["/fnexecd", "-addr", "0.0.0.0:8080"]
