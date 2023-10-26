FROM golang:1.21.3-alpine3.18

WORKDIR /app

COPY . .

RUN go mod tidy

ENV GO_ADDR=":5005"
ENTRYPOINT [ "go", "run", "." ]

EXPOSE 5005