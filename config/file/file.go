package file

import (
	"os"
	"regexp"
)

func GetRootPath() string {
	projectDirName := "distributed-sockets"
	projectName := regexp.MustCompile(`^(.*` + projectDirName + `)`)
	currentWorkDirectory, _ := os.Getwd()
	return string(projectName.Find([]byte(currentWorkDirectory)))
}
