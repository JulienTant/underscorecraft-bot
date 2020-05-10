FROM golang:alpine AS build-env

WORKDIR /myapp
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mc2d

# final stage
FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-env /myapp/mc2d .
ENTRYPOINT ["/app/mc2d"]