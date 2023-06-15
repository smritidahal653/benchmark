FROM golang:1.20 as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

COPY . .

RUN go mod download
RUN GO111MODULE=on CGO_ENABLED=0 go build -o pod-executor main.go

FROM  gcr.io/distroless/static
WORKDIR /
COPY --from=builder  /workspace/pod-executor .

ENTRYPOINT ["/pod-executor"]