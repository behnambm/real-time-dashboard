FROM golang:1.20
LABEL authors="behnam"

RUN mkdir /app
WORKDIR /app

COPY ./datasource.out .

CMD ["/app/datasource.out", "--server"]
