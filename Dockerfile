FROM hub.pengbei.tech:18080/bigdata/golang:1.19 as builder


USER root

WORKDIR /root/
COPY . /root/
RUN export CGO_ENABLED=0  \
    && export GOOS=linux  \
    && export GOARCH=amd64 \
    && export GOPROXY=https://goproxy.cn \
    && go install \
    && go build


FROM hub.pengbei.tech:18080/bigdata/alpine
RUN sed -i "s/dl-cdn\.alpinelinux\.org/mirrors\.aliyun\.com/g" /etc/apk/repositories
RUN apk add -U tzdata
RUN ls /usr/share/zoneinfo

ENV TZ=Asia/Shanghai

RUN mkdir /data
WORKDIR /data

COPY --from=builder /root/redis-proxy .
COPY ipipfree.ipdb /data/ipipfree.ipdb
ENV DEBUG=false
EXPOSE 8080
CMD ["./redis-proxy"]