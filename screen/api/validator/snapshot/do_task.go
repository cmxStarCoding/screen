package snapshot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
	"reflect"
)

type DoTaskRequest struct {
	JobNo  			string 		`json:"job_no" form:"job_no" validate:"required" comment:"截图任务编号"`
	BeginTime 		string 		`json:"begin_time" form:"begin_time" validate:"required" comment:"开始时间"`
	EndTime 		string 		`json:"end_time" form:"end_time" validate:"required" comment:"结束时间"`
	Type 			string 		`json:"type" form:"type" validate:"required" comment:"截图类型"`
	MediaType 		uint 		`json:"media_type" form:"media_type" validate:"required" comment:"媒体类型"`
	AdvertiserIds 	[]string 	`json:"advertiser_ids" form:"advertiser_ids" validate:"required" comment:"媒体账号"`
	BusinessType    string		`json:"business_type" form:"business_type" validate:"required" comment:"业务类型"`
}

func ValidateDoTaskRequest(c *gin.Context) (*DoTaskRequest, error) {
	var request DoTaskRequest
	utTrans := c.Value("Trans").(ut.Translator)
	Validate, _ := c.Get("Validate")
	validatorInstance, _ := Validate.(*validator.Validate)
	if err := c.ShouldBindJSON(&request); err != nil {
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
