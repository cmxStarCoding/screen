package controllers

import (
	"crm.com/common/cache"
	"crm.com/screen/api/services"
	"crm.com/screen/api/validator/snapshot"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

// SnapShotController 截图任务
type SnapShotController struct{}

// DoTask 截图任务
func (c SnapShotController) DoTask(ctx *gin.Context) {
	request, err := snapshot.ValidateDoTaskRequest(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.SnapshotService{}.DoTask(request)

	// 返回JSON数据
	ctx.JSON(200, "ok")
}

// CronTask 截图任务
func (c SnapShotController) CronTask(ctx *gin.Context) {
	request, err := snapshot.ValidateCronTaskRequest(ctx)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	cacheResult, _ := cache.RedisClient.Get("cron-crm-lock-" + now.Format("2006-01-02")).Result()
	if cacheResult == "1" {
		ctx.JSON(http.StatusBadRequest, "脚本正在运行中，请稍后再执行")
		return
	}

	go func() {
		services.SnapshotService{}.CronSnapTask(request)
	}()

	// 返回JSON数据
	ctx.JSON(200, "ok")
}
