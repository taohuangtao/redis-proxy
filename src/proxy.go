package src

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"redis-proxy/src/config"
	"sync"
	"time"
)

type RedisInfo struct {
	Master_host string
	Host_list   []string
	Password    string
}

var redisInfo = RedisInfo{}

var rwLock sync.RWMutex

func handleConnection(clientConn net.Conn, serverAddr string) {
	defer clientConn.Close()

	if config.DEBUG {
		logger.Println("DEBUG", "new connection")
	}

	// Connect to the server
	serverConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		logger.Println("Error connecting to server:", err.Error())
		return
	}
	defer serverConn.Close()

	// Copy data from client to server and vice versa

	go func() {
		defer clientConn.Close()
		defer serverConn.Close()
		io.Copy(serverConn, clientConn) // client -> server
	}()
	io.Copy(clientConn, serverConn) // server -> client

	if config.DEBUG {
		logger.Println("DEBUG", "connection closed")
	}
}

func checkMaster() {
	redisInfo.Master_host = GetMaster(redisInfo.Host_list, redisInfo.Password)

	// 定时执行master检查
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if config.DEBUG {
			logger.Println("DEBUG", "checking master")
		}
		rwLock.Lock() // 写锁
		redisInfo.Master_host = GetMaster(redisInfo.Host_list, redisInfo.Password)
		rwLock.Unlock()
	}
}

func getMaster() string {
	rwLock.RLock()
	defer rwLock.RUnlock()
	return redisInfo.Master_host
}

func RunInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	rwLock.RLock()
	defer rwLock.RUnlock()
	var info = make(map[string]interface{})
	info["Master_host"] = redisInfo.Master_host
	info["Host_list"] = redisInfo.Host_list
	b, _ := json.Marshal(info)
	w.Write(b)
}

func RunProxy(local_addr string, host_list []string, password string) {
	// Define the local address to listen on and the remote server address
	localAddr := local_addr //"127.0.0.1:8080"
	redisInfo.Host_list = host_list
	redisInfo.Password = password
	// 启动master检查
	go checkMaster()
	// 启动redis sentinel
	if config.RUN_REDIS_SENTINEL {
		go RunSentinel()
	}

	// Listen for incoming connections
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		logger.Println("ERROR", "Error starting TCP server:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	logger.Println("TCP Proxy listening on", localAddr)

	// 暴露信息
	http.HandleFunc("/debug/proxy/runinfo", RunInfo)

	for {
		// Wait for a connection
		clientConn, err := listener.Accept()
		if err != nil {
			logger.Println("ERROR", " accepting connection:", err.Error())
			continue
		}
		remoteAddr := getMaster()
		// Handle the connection in a new goroutine
		go handleConnection(clientConn, remoteAddr)
	}
}
