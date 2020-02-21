FROM golang:1.10-alpine as build

WORKDIR /go/src/github.com/AliyunContainerService/gpushare-scheduler-extender
COPY . .

RUN go build -o /go/bin/gpushare-sche-extender cmd/*.go

FROM alpine

COPY --from=build /go/bin/gpushare-sche-extender /usr/bin/gpushare-sche-extender

CMD ["gpushare-sche-extender"]
