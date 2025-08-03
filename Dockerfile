FROM golang:1.24.3-alpine3.21

WORKDIR /app

COPY . .

RUN go mod tidy && \
    cd cmd/reddit-migrate && \
    go build . && \
    mv reddit-migrate ../../

ENV GO_ADDR=":5005"
ENTRYPOINT ["./reddit-migrate"]

EXPOSE 5005
