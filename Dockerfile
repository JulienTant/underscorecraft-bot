FROM golang:alpine AS build-env

WORKDIR /myapp
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mc2d

# final stage
FROM alpine
RUN apk add --no-cache --update curl ca-certificates openssl git tar bash sqlite fontconfig \

RUN apk add --no-cache --update ca-certificates \
    && adduser --disabled-password --home /home/container container

USER container
ENV  USER=container HOME=/home/container

WORKDIR /home/container

COPY ./pterodactyl_entrypoint.sh /entrypoint.sh
RUN chmod +x entrypoint.sh
COPY --from=build-env /myapp/mc2d .
CMD ["/bin/bash", "/entrypoint.sh"]
