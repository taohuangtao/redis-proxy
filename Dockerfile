FROM golang:1.23 as builder


USER root

WORKDIR /root/
COPY go.mod /root/go.mod
COPY go.sum /root/go.sum
RUN cat /root/go.mod
RUN export CGO_ENABLED=0  \
    && export GOOS=linux  \
    && export GOARCH=amd64 \
    && export GOPROXY=https://goproxy.cn \
    && go mod download 

COPY . /root/
RUN go build


FROM alpine
RUN sed -i "s/dl-cdn\.alpinelinux\.org/mirrors\.aliyun\.com/g" /etc/apk/repositories
RUN apk add -U tzdata
RUN ls /usr/share/zoneinfo

ENV TZ=Asia/Shanghai

RUN mkdir /data
WORKDIR /data

COPY --from=builder /root/redis-proxy .
ENV DEBUG=false
ENV LOCAL_ADDR=127.0.0.1:6379
EXPOSE 6379
CMD ["./redis-proxy"]