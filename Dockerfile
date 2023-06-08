FROM golang:1.20 as builder

WORKDIR /workspace
COPY . .

RUN go build -o pod-executor .

CMD ["./pod-executor"]