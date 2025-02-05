package src

import (
	"context"
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
	path := "/redis_ht_sentinel"
	exists, _, err := t.zk.Exists(path)
	if err != nil {
		logger.Fatalf("failed to check existence of node %s: %v", path, err)
	}
	if !exists {
		t.zk.Create(path, []byte{}, 0, zk.WorldACL(zk.PermAll))
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
		logger.Printf("Failed to create lock node: %v", err)
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
	// 检查zk上一次检测存储的master记录，如果还是master就保持不变
	last_master, _, err := t.zk.Get("/redis_ht_sentinel/last_master")
	var master_host string
	if err == nil {
		_master_host := string(last_master)
		if config.DEBUG {
			logger.Printf("_master_host: %v", _master_host)
		}

		if _master_host != "" {
			rdb := redis.NewClient(&redis.Options{
				Addr:     _master_host,
				Password: t.PASSWORD,
				DB:       0,
				PoolSize: 1, // 禁用连接池，在这里无意义
			})
			defer rdb.Close()
			info, err := rdb.Info(context.Background(), "replication").Result()
			if err != nil {
				logger.Printf("Connection error, skipping %s err: %s", _master_host, err)
			} else {
				if config.DEBUG {
					logger.Printf("master replication: %v", info)
				}
				if strings.Contains(info, "role:master") {
					// 还是master
					master_host = _master_host
				}
			}
		}

	} else if err != zk.ErrNoNode {
		logger.Printf("Failed to get last master data: %v", err)
	}

	// 检测所有几点是
	var slaveList []string
	for _, redis_host := range t.REDIS_LIST {
		rdb := redis.NewClient(&redis.Options{
			Addr:     redis_host,
			Password: t.PASSWORD,
			DB:       0,
			PoolSize: 1, // 禁用连接池，在这里无意义
		})
		defer rdb.Close()
		info, err := rdb.Info(context.Background(), "replication").Result()
		if err != nil {
			logger.Printf("Connection error, skipping %s", redis_host)
			continue
		}
		if config.DEBUG {
			logger.Printf("replication %s", info)
		}

		// 是role:master
		if strings.Contains(info, "role:master") {
			if config.DEBUG {
				logger.Printf("is master %s", redis_host)
			}

			if master_host == "" {
				// 上一轮没有master,把这个设置为master
				master_host = redis_host
			} else if master_host != redis_host {
				// # 已经有一个master,但当前也是master,说明主从已经断开，将当前设置为slave 重新连上master
				logger.Printf("连上master[ slave: %s master: %s ]", redis_host, master_host)
				masterInfo := strings.Split(master_host, ":")
				rdb.SlaveOf(context.Background(), masterInfo[0], masterInfo[1])
			}
		} else {
			if config.DEBUG {
				logger.Printf("is slave %s", redis_host)
			}
			slaveList = append(slaveList, redis_host)
		}
	}

	if master_host == "" {
		logger.Println("-ERROR-ERROR-ERROR---- 没有找到 master -----ERROR-ERROR-ERROR-")
		logger.Println("-ERROR-ERROR-ERROR---- 重新将第一个slave 标记为master -----ERROR-ERROR-ERROR-")
		master_host = slaveList[0]
		slaveList = slaveList[1:]
		rdb := redis.NewClient(&redis.Options{
			Addr:     master_host,
			Password: t.PASSWORD,
			DB:       0,
			PoolSize: 1, // 禁用连接池，在这里无意义
		})
		defer rdb.Close()
		rdb.SlaveOf(context.Background(), "NO", "ONE")
	}

	if config.DEBUG {
		logger.Printf("-------- master %s -------", master_host)
	}
	// 存储最终master
	_, err = t.zk.Create("/redis_ht_sentinel/last_master", []byte(master_host), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		if err == zk.ErrNodeExists {
			_, err = t.zk.Set("/redis_ht_sentinel/last_master", []byte(master_host), -1)
			if err != nil {
				logger.Fatalf("ERROR Failed to set last master data: %v", err)
			}
		} else {
			logger.Fatalf("ERROR Failed to create last master data: %v", err)
		}
	}

	// 所有slave全部重新连上master
	for _, slave_host := range slaveList {
		rdb := redis.NewClient(&redis.Options{
			Addr:     slave_host,
			Password: t.PASSWORD,
			DB:       0,
			PoolSize: 1, // 禁用连接池，在这里无意义
		})
		defer rdb.Close()
		info, err := rdb.Info(context.Background(), "replication").Result()
		if err != nil {
			logger.Printf("ERROR Connection error, skipping %s", slave_host)
			continue
		}
		if config.DEBUG {
			logger.Println(info)
		}

		masterInfo := strings.Split(master_host, ":")
		if !strings.Contains(info, fmt.Sprintf("master_host:%s", masterInfo[0])) || !strings.Contains(info, fmt.Sprintf("master_port:%s", masterInfo[1])) {
			logger.Printf("重新连上 master[ slave: %s master: %s ]", slave_host, master_host)
			rdb.SlaveOf(context.Background(), masterInfo[0], masterInfo[1])
		}

	}
}

func RunSentinel() {
	for {
		if config.DEBUG {
			logger.Println("start")
		}
		task := newTask()
		task.run()
		task.clear()
		if config.DEBUG {
			logger.Println("睡眠随机时间")
		}
		time.Sleep(time.Duration(1+rand.Float64()*10) * time.Second)
	}
}
