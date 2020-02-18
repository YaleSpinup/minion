# build stage
FROM golang:alpine AS build-env

ARG version="0.0.0"
ARG prerelease=""
ARG githash=""
ARG buildstamp=""

RUN apk add --no-cache git openssh-client gcc musl-dev
RUN mkdir /app
WORKDIR /app
RUN go version
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o /app/api.out  -ldflags="-X main.Version=$version -X main.VersionPrerelease=$prerelease -X main.githash=$githash -X main.buildstamp=$buildstamp" *.go
RUN /app/api.out -version

# final stage
FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-env /app/api.out /app/api
EXPOSE 80
ENTRYPOINT ["./api"]
