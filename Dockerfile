FROM golang:1.22-alpine AS builder

WORKDIR /builder

COPY go.mod go.sum ./

RUN apk update && apk upgrade --no-cache && apk add --no-cache ca-certificates && go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM scratch

COPY --from=builder /build/main /go/bin/main

EXPOSE 8080

ENTRYPOINT [ "go/bin/main" ]