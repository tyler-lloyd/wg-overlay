FROM golang:1.17-alpine

WORKDIR /go/src/wg-overlay

COPY go.mod ./
COPY go.sum ./

COPY cmd/wg-node-watcher/main.go ./
COPY pkg/ ./pkg/

RUN go build -o /wireguard-network-service

CMD [ "/wireguard-network-service" ]