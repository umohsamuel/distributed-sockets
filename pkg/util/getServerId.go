package util

import "os"

func GetServerID() string {
	id := os.Getenv("SERVER_ID")
	if id != "" {
		return id
	}
	hostname, _ := os.Hostname()
	return hostname
}
