# Following the example from https://docs.docker.com/language/golang/build-images/.
FROM golang:1.21
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /main
RUN echo $GOOGLE_APPLICATION_CREDENTIALS > /tmp/google_application_credentials.json
EXPOSE 8080
ENV SENTRY_DSN=$SENTRY_DSN
ENV GOOGLE_APPLICATION_CREDENTIALS=/tmp/google_application_credentials.json
CMD ["/main"]
