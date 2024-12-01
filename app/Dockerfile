################################################################################
# parser
################################################################################
FROM golang:alpine as modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

FROM golang:alpine as builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
ENV  CONFIG_PATH=./config/config.yaml
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin/parser ./cmd/main.go
CMD ["/bin/parser"]
