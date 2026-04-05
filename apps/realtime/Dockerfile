FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/realtime main.go

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/realtime /realtime
EXPOSE 8081
ENTRYPOINT ["/realtime"]
