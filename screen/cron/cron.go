package cron

import (
	"crm.com/screen/api/services"
	"crm.com/screen/api/validator/snapshot"
	"fmt"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
)

func RegisterCron() {
	// 创建 cron 实例
	c := cron.New()
	//c := cron.New(cron.WithSeconds()) //秒级别的定时器，只能运行秒级的

	// 添加定时任务，每天凌晨3点执行
	_, _ = c.AddFunc("0 3 * * *", func() {
		// 在这里执行你的脚本或任务
		services.SnapshotService{}.CronSnapTask(&snapshot.CronTaskRequest{
			CronDate: strconv.Itoa(time.Now().Day()),
		})
	})
	// 每五分钟执行一次
	_, _ = c.AddFunc("*/5 * * * *", func() {
		// 在这里执行你的脚本或任务
		//services.SnapshotService{}.CronSnapTask()
	})

	// 每五秒执行一次
	//_, _ = c.AddFunc("*/10 * * * * *", func() {
	// 在这里执行你的脚本或任务
	//runYourScript1()
	//services.SnapshotService{}.CronSnapTask()
	//})

	// 输出日志，确保 cron 任务被注册
	fmt.Println("Cron tasks registered")
	// 启动 cron
	c.Start()
}
