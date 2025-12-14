FROM golang:1.25.5-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download
RUN go get github.com/jackc/puddle/v2

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o death-level-tracker ./cmd/bot

FROM alpine:latest

WORKDIR /root/

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/death-level-tracker .

CMD ["./death-level-tracker"]