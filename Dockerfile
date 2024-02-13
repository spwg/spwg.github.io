FROM golang:1.22 as build

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app main.go

FROM debian:12
COPY --from=build /usr/local/bin/app /usr/local/bin/app
CMD ["app"]
