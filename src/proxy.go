package src

import (
	"io"
	"log"
	"net"
	"os"
	"redis-proxy/src/config"
	"sync"
	"time"
)

type RedisInfo struct {
	master_host string
	host_list   []string
	password    string
}

var redisInfo = RedisInfo{}

var rwLock sync.RWMutex

func handleConnection(clientConn net.Conn, serverAddr string) {
	defer clientConn.Close()

	if config.DEBUG {
		log.Println("DEBUG", "new connection")
	}

	// Connect to the server
	serverConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Println("Error connecting to server:", err.Error())
		return
	}
	defer serverConn.Close()

	// Copy data from client to server and vice versa
	go io.Copy(serverConn, clientConn) // client -> server
	io.Copy(clientConn, serverConn)    // server -> client

	if config.DEBUG {
		log.Println("DEBUG", "connection closed")
	}
}

func checkMaster() {
	redisInfo.master_host = GetMaster(redisInfo.host_list, redisInfo.password)

	// 定时执行master检查
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if config.DEBUG {
			log.Println("DEBUG", "checking master")
		}
		rwLock.Lock() // 写锁
		redisInfo.master_host = GetMaster(redisInfo.host_list, redisInfo.password)
		rwLock.Unlock()
	}
}

func getMaster() string {
	rwLock.RLock()
	defer rwLock.RUnlock()
	return redisInfo.master_host
}

func RunProxy(local_addr string, host_list []string, password string) {
	// Define the local address to listen on and the remote server address
	localAddr := local_addr //"127.0.0.1:8080"
	redisInfo.host_list = host_list
	redisInfo.password = password
	go checkMaster()

	// Listen for incoming connections
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		log.Println("ERROR", "Error starting TCP server:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	log.Println("TCP Proxy listening on", localAddr)

	for {
		// Wait for a connection
		clientConn, err := listener.Accept()
		if err != nil {
			log.Println("ERROR", " accepting connection:", err.Error())
			continue
		}
		remoteAddr := getMaster()
		// Handle the connection in a new goroutine
		go handleConnection(clientConn, remoteAddr)
	}
}
