package main

import (
	"log"
	"redis-proxy/src"
	"strings"
)

func main() {
	src.RunProxy("127.0.0.1:28080", strings.Split("rds.internal.pengbei.net:6379", ","), "rds_PWD")
	log.Println("DEBUG", "master:")
}
