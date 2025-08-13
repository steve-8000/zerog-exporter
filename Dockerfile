FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o zerog-exporter .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/zerog-exporter .
COPY config.yml ./
EXPOSE 26330
ENTRYPOINT ["./zerog-exporter"]