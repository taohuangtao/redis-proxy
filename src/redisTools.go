package src

import (
	"context"
	"fmt"
	"redis-proxy/src/config"
	"strings"

	"github.com/go-redis/redis/v8"
)

var DEBUG bool = config.DEBUG

func check(host string, password string) bool {
	// 创建一个上下文
	ctx := context.Background()

	// 创建一个Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     host,     // Redis服务器地址
		Password: password, // 如果没有密码则留空
		DB:       0,        // 使用默认数据库
	})
	defer rdb.Close()

	// 测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Fatalf("Error connecting to Redis: %v Host: %v", err, host)
	}
	if DEBUG {
		logger.Println("DEBUG", host+"Connected to Redis")
	}

	// 检查是否为主节点
	isMaster, err := isRedisMaster(ctx, rdb)
	if err != nil {
		logger.Fatalf("Error checking Redis role: %v", err)
	}
	if DEBUG {
		if isMaster {
			logger.Println("DEBUG", host+" is a master.")
		} else {
			logger.Println("DEBUG", host+" is not a master.")
		}
	}

	return isMaster
}

func GetMaster(host_list []string, password string) string {
	count := len(host_list)
	for i := 0; i < count; i++ {
		host := host_list[i]
		if check(host, password) {
			if DEBUG {
				logger.Println("DEBUG", host+" is a master.")
			}
			return host
		}
	}
	return ""
}

func isRedisMaster(ctx context.Context, client *redis.Client) (bool, error) {
	// INFO 命令获取关于 Redis 服务器的各种信息和统计
	info, err := client.Info(ctx, "replication").Result()

	if DEBUG {
		// log.Println("DEBUG", "info: "+info)
	}
	if err != nil {
		return false, fmt.Errorf("failed to get info: %v", err)
	}

	// 检查info字符串中是否包含 "role:master"
	return containsRoleMaster(info), nil
}

func containsRoleMaster(info string) bool {
	const roleMaster = "role:master"
	return strings.Contains(info, roleMaster)
}
