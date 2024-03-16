package main

import (
	"crm.com/common/cache"
	"crm.com/common/database"
	"crm.com/common/middleware"
	"crm.com/common/utils"
	"crm.com/screen/api/v1"
	"crm.com/screen/cron"
	"crm.com/screen/queue"
	"flag"
	"github.com/gin-gonic/gin"
)

func main() {
	var env string
	flag.StringVar(&env, "env", "local", "设置环境")
	// 解析启动的命令行参数
	flag.Parse()
	if env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	// 初始化Gin
	r := gin.Default()
	//加载配置文件
	// 为 multipart forms 设置较低的内存限制 (默认是 32 MiB)
	r.MaxMultipartMemory = 2 << 20 // 8 MiB
	//允许跨域
	r.Use(middleware.CORSMiddleware())
	//初始化日志文件
	utils.SetupLogger()
	// 初始化数据库连接
	database.InitCrmDB()
	// 初始化数据库连接
	database.InitCmsDB()
	// 初始化数据库连接
	database.InitMapiReportDB()
	// 初始化redis链接
	cache.InitRedisClient()
	// 设置API路由
	v1.SetupRoutes(r)
	//静态资源配置
	r.Static("/static", "../static")
	//注册rabbitmq消费者
	queue.RegisterRabbitMQConsumer()
	//初始化定时任务
	cron.RegisterCron()
	// 启动服务
	r.Run(":8083")
}
