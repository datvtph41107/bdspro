FROM golang:1.27.1-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -o /out/bdspro .

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
USER app
WORKDIR /app
COPY --from=builder /out/bdspro /app/bdspro
COPY app.env /app/app.env
EXPOSE 8080
CMD ["/app/bdspro"]
