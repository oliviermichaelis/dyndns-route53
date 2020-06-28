# build stage
FROM golang:alpine AS build-env

RUN apk --no-cache add build-base git linux-headers
WORKDIR /go/src/github.com/oliviermichaelis/dyndns-route53
COPY . .
RUN go install -ldflags="-s -w" -v ./...

FROM alpine

RUN addgroup -S dyndns && adduser -S dyndns -G dyndns
USER dyndns

WORKDIR /app
COPY --from=build-env /go/bin/dyndns-route53 /app/

ENTRYPOINT ./dyndns-route53