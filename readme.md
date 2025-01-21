# 基于golang的redis代理

1、目前支持自动检查主从模式，将所有流量转发到主节点  
2、自带一个redis哨兵，只需要部署redis多实例后，哨兵制动选出master节点，其他节点自动连接到master节点

## 配置信息，通过环境变量配置
```
LOCAL_ADDR=127.0.0.1:28080 # 本地监听的端口
REDIS_HOSTS=192.168.1.11:6379,192.168.1.12:6379 # redis地址，多个用逗号分隔
REDIS_PASSWORD=123456 # redis密码
ZK_HOSTS=192.168.1.21:2181,192.168.1.22:2181,192.168.1.23:2181 # zookeeper地址，多个用逗号分隔
RUN_REDIS_SENTINEL=true # 是否使用redis哨兵
```