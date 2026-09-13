FROM golang:1.25-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install -tags 'postgres file' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1

COPY . .
RUN go build -o /app/bin/api ./cmd/api

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/api ./api
COPY --from=builder /go/bin/migrate ./migrate
COPY migrations ./migrations

EXPOSE 8080

CMD ["./api"]
