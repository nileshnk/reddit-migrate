FROM golang:1.21.3-alpine3.18

WORKDIR /app

COPY . .

RUN go mod tidy

ENV HOST=":5000"
ENTRYPOINT [ "go", "run", "." ]

EXPOSE 5005