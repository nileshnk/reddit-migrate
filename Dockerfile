FROM golang:1.24.3-alpine3.21

WORKDIR /app

COPY . .

RUN go mod tidy

ENV GO_ADDR=":5005"
ENTRYPOINT [ "go", "run", "." ]

EXPOSE 5005