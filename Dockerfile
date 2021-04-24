FROM golang:alpine AS build-env

WORKDIR /myapp
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mc2d

# final stage
FROM alpine

COPY ./pterodactyl_entrypoint.sh /entrypoint.sh

RUN apk add --no-cache --update ca-certificates bash \
    && adduser --disabled-password --home /home/container container \
    && chmod a+x /entrypoint.sh

USER container
ENV  USER=container HOME=/home/container
WORKDIR /home/container

COPY --from=build-env /myapp/mc2d .
CMD ["/bin/bash", "/entrypoint.sh"]
