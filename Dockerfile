FROM ubuntu as builder


USER root

WORKDIR /root/
RUN cat /etc/os-release
RUN echo 'deb https://mirrors.aliyun.com/ubuntu/ noble main restricted universe multiverse \n\
deb-src https://mirrors.aliyun.com/ubuntu/ noble main restricted universe multiverse \n\
 \n\
deb https://mirrors.aliyun.com/ubuntu/ noble-security main restricted universe multiverse \n\
deb-src https://mirrors.aliyun.com/ubuntu/ noble-security main restricted universe multiverse \n\
\n\
deb https://mirrors.aliyun.com/ubuntu/ noble-updates main restricted universe multiverse\n\
deb-src https://mirrors.aliyun.com/ubuntu/ noble-updates main restricted universe multiverse\n\
\n\
# deb https://mirrors.aliyun.com/ubuntu/ noble-proposed main restricted universe multiverse\n\
# deb-src https://mirrors.aliyun.com/ubuntu/ noble-proposed main restricted universe multiverse\n\
\n\
deb https://mirrors.aliyun.com/ubuntu/ noble-backports main restricted universe multiverse\n\
deb-src https://mirrors.aliyun.com/ubuntu/ noble-backports main restricted universe multiverse\n\
' > /etc/apt/sources.list 
RUN cat /etc/apt/sources.list 
RUN apt-get update
RUN apt-get install -y wget
RUN wget https://golang.google.cn/dl/go1.23.5.linux-amd64.tar.gz
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
RUN echo $PATH
RUN chmod +x -R /usr/local/go/bin
RUN ls -lh /usr/local/go/bin
RUN go version
COPY go.mod /root/go.mod
COPY go.sum /root/go.sum
RUN export CGO_ENABLED=0  \
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
# COPY ./redis-proxy .
RUN chmod +x redis-proxy
ENV DEBUG=false
ENV LOCAL_ADDR=127.0.0.1:6379
EXPOSE 6379
CMD ["./redis-proxy"]