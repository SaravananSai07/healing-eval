FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

FROM alpine:3.19 AS server

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /server /server

EXPOSE 8080

CMD ["/server"]

FROM alpine:3.19 AS worker

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /worker /worker

CMD ["/worker"]

