package snapshot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
	"reflect"
)

type CronTaskRequest struct {
	CronDate string `json:"cron_date" form:"cron_date" validate:"required" comment:"日期"`
}

func ValidateCronTaskRequest(c *gin.Context) (*CronTaskRequest, error) {
	var request CronTaskRequest
	utTrans := c.Value("Trans").(ut.Translator)
	Validate, _ := c.Get("Validate")
	validatorInstance, _ := Validate.(*validator.Validate)
	if err := c.ShouldBindQuery(&request); err != nil {
		return nil, err
	}
	// 收集结构体中的comment标签，用于替换英文字段名称，这样返回错误就能展示中文字段名称了
	validatorInstance.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return fld.Tag.Get("comment")
	})
	// 进行进一步的验证
	err := validatorInstance.Struct(request) //这里的err是未翻译之前的
	if err != nil {
		errs := err.(validator.ValidationErrors)
		var sliceErrs []string
		for _, e := range errs {
			sliceErrs = append(sliceErrs, e.Translate(utTrans))
		}
		return nil, fmt.Errorf(sliceErrs[0])
	}

	return &request, nil
}
