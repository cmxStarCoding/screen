package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
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

// 生成唯一的随机字符串
func GenerateUniqueRandomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomString := make([]byte, 10) // 10个字符长度，你可以根据需要调整
	rand.Seed(time.Now().UnixNano())
	for i := range randomString {
		randomString[i] = charset[rand.Intn(len(charset))]
	}
	// 使用时间戳生成唯一性
	uniquePart := time.Now().UnixNano()
	return fmt.Sprintf("%s%d", string(randomString), uniquePart)
}

func StringNumericToUnit(stringNumeric string) uint {
	id64, _ := strconv.ParseUint(stringNumeric, 10, 64)
	return uint(id64)
}

func ContainValue(slice []uint64, value uint64) bool {

	for _, v := range slice {

		if v == value {
			return true
		}
	}
	return false
}

func CreateLogDir(dirName string) {
	//创建日志文件夹
	//dirName := ".log/product_flow_snapshot/" + lastMonth + "/" + strconv.Itoa(int(productConfigList[i].ID))
	//err := os.MkdirAll(dirName, 0755)
	//if err != nil {
	//	log.Println("创建文件夹"+dirName+"出错:", err)
	//	continue
	//}
	//logFile, logErr := os.OpenFile("output.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	//if logErr != nil {
	//	gin.DefaultWriter.Write([]byte("月度流水创建日志文件错误"))
	//}
	//log.SetOutput(logFile)
	//log.Println("111111")
	//defer logFile.Close()

	err := os.MkdirAll(dirName, 0755)
	if err != nil {
		log.Fatal("创建日志文件夹失败")
	}

}
