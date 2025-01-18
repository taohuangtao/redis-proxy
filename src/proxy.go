package src

import (
	"fmt"
	"io"
	"net"
	"os"
	"redis-proxy/src/config"
	"time"
)

type RedisInfo struct {
	master_host string
	host_list   []string
	password    string
}

var redisInfo = RedisInfo{}

func handleConnection(clientConn net.Conn, serverAddr string) {
	defer clientConn.Close()

	// Connect to the server
	serverConn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Println("Error connecting to server:", err.Error())
		return
	}
	defer serverConn.Close()

	// Copy data from client to server and vice versa
	go io.Copy(serverConn, clientConn) // client -> server
	io.Copy(clientConn, serverConn)    // server -> client
}

func checkMaster() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if config.DEBUG {
			fmt.Println("checking master")
		}
		redisInfo.master_host = GetMaster(redisInfo.host_list, redisInfo.password)
	}
}

func RunProxy(local_addr string, host_list []string, password string) {
	// Define the local address to listen on and the remote server address
	localAddr := local_addr //"127.0.0.1:8080"
	redisInfo.host_list = host_list
	redisInfo.password = password
	checkMaster()

	// Listen for incoming connections
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		fmt.Println("Error starting TCP server:", err.Error())
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Println("TCP Proxy listening on", localAddr)

	for {
		// Wait for a connection
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err.Error())
			continue
		}
		remoteAddr := redisInfo.master_host
		// Handle the connection in a new goroutine
		go handleConnection(clientConn, remoteAddr)
	}
}
