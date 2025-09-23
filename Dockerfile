FROM golang:1.24.2-alpine AS build

RUN apk add --no-cache git ca-certificates build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /savanna-api ./main.go

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S -G app app
USER app

COPY --from=build /savanna-api /savanna-api
EXPOSE 8080

ENV PORT=8080
CMD ["/savanna-api"]
