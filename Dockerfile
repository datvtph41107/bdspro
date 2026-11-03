# Build stage
FROM golang:1.27.3-alpine3.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

# Run stage
FROM alpine:3.24
WORKDIR /app
COPY --from=builder /app/main .
COPY docker/root/etc/templates/app.ini /etc/templates/app.ini
COPY docker/setup /app/setup
COPY wait-for.sh .
COPY db/migration ./db/migration

RUN chmod +x /app/wait-for.sh /app/setup

EXPOSE 8080
CMD ["/app/main"]
ENTRYPOINT ["/app/setup"]
