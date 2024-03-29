package utils

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"os"
)

func SetupLogger() {
	//设置日志文件
	gin.DisableConsoleColor()
	// 记录到文件。
	f, _ := os.Create("gin.log")
	//gin.DefaultWriter = io.MultiWriter(f)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
}

func Md5Hash(input string) string {
	md5Hash := md5.New()
	md5Hash.Write([]byte(input))
	return hex.EncodeToString(md5Hash.Sum(nil))
}

func CreateDir(dirName string) {

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		log.Fatal("创建文件夹失败")
	}

}
