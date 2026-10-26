FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/proxy ./main.go

FROM alpine:3.20
RUN adduser -D -H -u 10001 appuser
USER appuser
COPY --from=build /out/proxy /proxy
ENTRYPOINT ["/proxy"]
