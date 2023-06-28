FROM golang:1.20
LABEL authors="behnam"

RUN mkdir /app
WORKDIR /app

COPY ./server.out .
COPY ./templates ./templates

EXPOSE 8080

CMD ["/app/server.out"]
