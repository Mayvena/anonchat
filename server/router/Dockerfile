FROM golang:1.22-alpine AS build
WORKDIR /src
COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/router ./main.go

FROM alpine:3.20
RUN adduser -D -H -u 10002 appuser
USER appuser
COPY --from=build /out/router /router
ENTRYPOINT ["/router"]
