FROM golang:1.17-alpine

WORKDIR /go/src/wg-overlay

COPY . .

RUN go build -o /wnc cmd/controller/main.go
RUN go build -o /wg-init cmd/initializer/main.go
