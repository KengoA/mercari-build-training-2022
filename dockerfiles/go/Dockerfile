FROM golang:1.18.0-alpine

WORKDIR /project
RUN apk add build-base

COPY db ./db/

COPY go/go.mod ./
COPY go/go.sum ./

RUN go mod download

COPY go/app/main.go ./
COPY go/images ./images/


RUN go build -o /mercari-build-training-app

EXPOSE 9000

CMD [ "/mercari-build-training-app" ]
