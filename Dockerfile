# Build stage
FROM golang:1.27.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/main main.go

# Run stage
FROM alpine:3.22
WORKDIR /app
COPY --from=builder /out/main /app/main
COPY app.env /app/app.env
COPY start.sh /app/start.sh
COPY wait-for.sh /app/wait-for.sh
COPY db/migration /app/db/migration
RUN chmod +x /app/start.sh /app/wait-for.sh

EXPOSE 8080 9090
ENTRYPOINT ["/app/start.sh"]
CMD ["/app/main"]
