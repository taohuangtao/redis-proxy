package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"redis-proxy/src"
	"redis-proxy/src/config"
)

func main() {

	go src.RunProxy(config.LOCAL_ADDR, config.REDIS_HOSTS, config.REDIS_PWD)

	// 定时监控协程数量
	// ticker := time.NewTicker(3 * time.Second)
	// defer ticker.Stop()

	// for range ticker.C {
	// 	numGoroutines := runtime.NumGoroutine()
	// 	log.Printf("Current number of goroutines: %d\n", numGoroutines)
	// }

	// 启动HTTP服务器
	func() {
		log.Println("Starting pprof server on :6060")
		http.ListenAndServe(":6060", nil)
	}()
}
