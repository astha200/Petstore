FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod ./
RUN go mod download && go mod tidy

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/petstore ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/petstore /app/petstore
EXPOSE 8080
ENTRYPOINT ["/app/petstore"]
