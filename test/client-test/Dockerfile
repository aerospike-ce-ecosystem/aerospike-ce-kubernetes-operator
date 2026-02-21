FROM golang:1.25 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o /client-test .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /client-test /client-test
ENTRYPOINT ["/client-test"]
