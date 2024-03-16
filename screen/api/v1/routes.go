// service1/api/v1/routes.go

package v1

import (
	"crm.com/common/middleware"
	"crm.com/screen/api/controllers"
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置API路由
func SetupRoutes(r *gin.Engine) {
	apiV1NoNeedLogin := r.Group("/api/v1").Use(middleware.TranslationsMiddleware())
	{
		//执行截图任务
		apiV1NoNeedLogin.POST("/do_snapshot_task", controllers.SnapShotController{}.DoTask)

		//执行截图任务
		apiV1NoNeedLogin.GET("/snapshot_cron_task", controllers.SnapShotController{}.CronTask)

	}
	//apiV1NeedLogin := r.Group("/api/v1").Use(middleware.TranslationsMiddleware(), middleware.JWTMiddleware())
	//{
	//
	//}
}
