package src

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"redis-proxy/src/config"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/samuel/go-zookeeper/zk"
)

var logger *log.Logger

func init() {
	if os.Getenv("DEBUG") == "true" {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}
}

type Task struct {
	ZK_HOSTS   []string
	zk         *zk.Conn
	REDIS_LIST []string
	PASSWORD   string
}

func newTask() *Task {
	t := &Task{
		ZK_HOSTS: config.ZK_HOSTS,
	}
	var err error
	t.zk, _, err = zk.Connect(t.ZK_HOSTS, time.Second*10)
	if err != nil {
		logger.Fatalf("Failed to connect to Zookeeper: %v", err)
	}

	t.REDIS_LIST = config.REDIS_HOSTS
	t.PASSWORD = config.REDIS_PWD

	return t
}

func (t *Task) lock() bool {
	_, err := t.zk.Create("/redis_ht_sentinel/task_lock", []byte{}, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists {
			return false
		}
		logger.Fatalf("Failed to create lock node: %v", err)
	}
	return true
}

func (t *Task) clear() {
	t.zk.Close()
}

func (t *Task) run() {
	if t.lock() {
		logger.Println("获得锁")
		t._run()
	} else {
		logger.Println("没有获得锁，离开")
	}
}

func (t *Task) _run() {
	var masterInfo = map[string]string{}
	data, _, err := t.zk.Get("/redis_ht_sentinel/last_master")
	if err == nil {
		err = json.Unmarshal(data, &masterInfo)
		if err != nil {
			logger.Printf("Failed to unmarshal last master data: %v", err)
		} else {
			host := masterInfo["host"]
			port := masterInfo["port"]
			rdb := redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%s", host, port),
				Password: t.PASSWORD,
				DB:       0,
			})
			info, err := rdb.Info(context.Background(), "replication").Result()
			if err != nil {
				logger.Printf("Connection error, skipping %s", host)
			} else {
				if strings.Contains(info, "role:master") {
					masterInfo = map[string]string{"host": host, "port": port}
				}
			}
		}
	} else if err != zk.ErrNoNode {
		logger.Printf("Failed to get last master data: %v", err)
	}

	var slaveList []map[string]string
	for _, redisHost := range t.REDIS_LIST {
		_redisHost := strings.Split(redisHost, ":")
		host := _redisHost[0]
		port := _redisHost[1]
		rdb := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: t.PASSWORD,
			DB:       0,
		})
		info, err := rdb.Info(context.Background(), "replication").Result()
		if err != nil {
			logger.Printf("Connection error, skipping %s", host)
			continue
		}
		logger.Printf("replication %s", info)
		if strings.Contains(info, "role:master") {
			logger.Printf("is master %s", host)
			if masterInfo == nil {
				masterInfo = map[string]string{"host": host, "port": port}
			} else if masterInfo["host"] != host || masterInfo["port"] != port {
				// # 已经有一个master,但当前也是master,说明主从已经断开，将当前设置为slave 重新连上master
				logger.Printf("连上master[ slave: %s master: %s ]", host, masterInfo["host"])
				rdb.SlaveOf(context.Background(), masterInfo["host"], masterInfo["port"])
			}
		} else {
			logger.Println("is slave")
			slaveList = append(slaveList, map[string]string{"host": host, "port": port})
		}
	}

	if masterInfo == nil {
		logger.Println("-ERROR-ERROR-ERROR---- 没有找到 master -----ERROR-ERROR-ERROR-")
		logger.Println("-ERROR-ERROR-ERROR---- 重新将第一个slave 标记为master -----ERROR-ERROR-ERROR-")
		masterInfo = slaveList[0]
		slaveList = slaveList[1:]
		rdb := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", masterInfo["host"], masterInfo["port"]),
			Password: t.PASSWORD,
			DB:       0,
		})
		rdb.SlaveOf(context.Background(), "NO", "ONE")
	}

	data, err = json.Marshal(masterInfo)
	if err != nil {
		logger.Fatalf("Failed to marshal master info: %v", err)
	}
	_, err = t.zk.Create("/redis_ht_sentinel/last_master", data, 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists {
			_, err = t.zk.Set("/redis_ht_sentinel/last_master", data, -1)
			if err != nil {
				logger.Fatalf("Failed to set last master data: %v", err)
			}
		} else {
			logger.Fatalf("Failed to create last master data: %v", err)
		}
	}

	for _, sInfo := range slaveList {
		host := sInfo["host"]
		port := sInfo["port"]
		rdb := redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: t.PASSWORD,
			DB:       0,
		})
		info, err := rdb.Info(context.Background(), "replication").Result()
		if err != nil {
			logger.Printf("Connection error, skipping %s", host)
			continue
		}
		if !strings.Contains(info, fmt.Sprintf("master_host:%s", masterInfo["host"])) {
			logger.Printf("连上master[ slave: %s master: %s ]", host, masterInfo["host"])
			rdb.SlaveOf(context.Background(), masterInfo["host"], masterInfo["port"])
		}
	}
}

func RunSentinel() {
	for {
		logger.Println("start")
		task := newTask()
		task.run()
		task.clear()
		logger.Println("睡眠随机时间")
		time.Sleep(time.Duration(1+rand.Float64()*10) * time.Second)
	}
}
