# 基于golang的redis代理

1、目前支持自动检查主从模式，将所有流量转发到主节点  
2、自带一个redis哨兵，只需要部署redis多实例后，哨兵制动选出master节点，其他节点自动连接到master节点

## 配置信息，通过环境变量配置
```
LOCAL_ADDR=127.0.0.1:28080 # 本地监听的端口
REDIS_HOSTS=rds.internal.pengbei.net:6379,rds.internal.pengbei.net:6379 # redis地址，多个用逗号分隔
REDIS_PASSWORD=123456 # redis密码
ZK_HOSTS=127.0.0.1:2181,127.0.0.1:2181 # zookeeper地址，多个用逗号分隔
RUN_REDIS_SENTINEL=true # 是否使用redis哨兵
```