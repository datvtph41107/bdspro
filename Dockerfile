# Lecture 23: a small, non-root runtime image; migrations run as a separate job.
FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bdspro .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/bdspro /app/bdspro
COPY app.env /app/app.env
USER app
EXPOSE 8080
ENTRYPOINT ["/app/bdspro"]
