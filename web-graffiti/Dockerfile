FROM golang:1.23.4-alpine as builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY ./src .

RUN go build -o main .

FROM alpine:latest

WORKDIR /

COPY --from=builder /app/main .

CMD ["./main"]