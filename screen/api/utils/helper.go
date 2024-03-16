package utils

import (
	"archive/zip"
	"crm.com/screen/library"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ConvertStringToIntSlice(str string) ([]int, error) {
	// 使用 strings.Split 函数拆分字符串
	strSlice := strings.Split(str, ",")

	// 初始化 int 类型的切片
	intSlice := make([]int, 0, len(strSlice))

	// 遍历字符串切片，并将每个字符串转化为整数
	for _, s := range strSlice {
		i, err := strconv.Atoi(s)
		if err != nil {
			// 处理转换错误
			return nil, err
		}
		intSlice = append(intSlice, i)
	}

	return intSlice, nil
}

// 获取上个月的月份
func GetLastMonth(format string) string {

	// 获取当前时间
	currentTime := time.Now()

	// 获取上个月的时间
	lastMonth := currentTime.AddDate(0, -1, 0)

	// 格式化输出上个月的月份
	lastMonthFormat := lastMonth.Format(format)
	return lastMonthFormat
}

func UploadLocalFile(localFilePath string, tosPAth string) (ossPath string, err error) {
	_, FileExistsErr := os.Stat(localFilePath)
	if os.IsNotExist(FileExistsErr) {
		log.Println("无法上传文件到tos,文件不存在,路径为:" + ossPath)
		return "", nil
	}
	file, err := os.Open(localFilePath)
	if err != nil {
		log.Println("无法打开文件,错误信息", err.Error())
		return "", err
	}
	oss, err := library.NewTos()
	if err == nil {
		if oss.Upload(file, tosPAth) == nil {
			return "https://cms-static.pengwin.com/" + tosPAth, nil
		}
	}
	return "", err
}

func CompressFolderZip(sourceDir string, targetZip string) (string, error) {
	// 源文件夹路径
	//sourceDir := "./a"
	// 压缩后的 zip 包路径
	//targetZip := "./output.zip"

	// 创建或覆盖目标 zip 文件
	zipFile, err := os.Create(targetZip)
	if err != nil {
		log.Println("创建zip压缩包失败，失败原因:", err)
		return "fail", err
	}
	defer zipFile.Close()

	// 创建 zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历源文件夹下的文件和子文件夹
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件夹则忽略
		if info.IsDir() {
			return nil
		}

		// 创建 zip 包内的文件
		fileInZip, err := zipWriter.Create(info.Name())
		if err != nil {
			return err
		}

		// 打开源文件
		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		// 将源文件内容复制到 zip 包内的文件
		_, err = io.Copy(fileInZip, sourceFile)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Println("压缩文件夹失败，失败原因:", err)
		return "", err
	}
	return "ok", nil
}
