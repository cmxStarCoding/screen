package controllers

import (
	"crm.com/screen/api/services"
	"crm.com/screen/api/validator/snapshot"
	"github.com/gin-gonic/gin"
	"net/http"
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
	services.SnapshotService{}.CronSnapTask(request)

	// 返回JSON数据
	ctx.JSON(200, "ok")
}
