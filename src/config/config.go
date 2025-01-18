package config

import (
	"os"
)

var DEBUG bool = GetDebug()

func GetDebug() bool {
	if os.Getenv("DEBUG") != "true" {
		return true
	} else {
		return false
	}
}
