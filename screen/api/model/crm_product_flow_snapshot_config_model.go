// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameCrmProductFlowSnapshotConfig = "crm_product_flow_snapshot_config"

// CrmProductFlowSnapshotConfig 产品流水月度快照配置表
type CrmProductFlowSnapshotConfig struct {
	ID           int32     	`gorm:"column:id;primaryKey;autoIncrement:true;comment:主键id" json:"id"` // 主键id
	ProductIds   string    	`gorm:"column:product_ids;comment:产品id" json:"product_ids"`             // 产品id
	SendDate     int32     	`gorm:"column:send_date;comment:每月的发送日期" json:"send_date"`              // 每月的发送日期
	FsUserID     string    	`gorm:"column:fs_user_id;comment:飞书用户id" json:"fs_user_id"`             // 飞书用户id
	CreateUserID int32     	`gorm:"column:create_user_id;comment:创建人id" json:"create_user_id"`      // 创建人id
	Status       int32     	`gorm:"column:status;comment:状态0关闭1开启，默认为0" json:"status"`              // 状态0关闭1开启，默认为0
	AddTime      TimeNormal `gorm:"column:add_time;autoCreateTime;comment:创建时间" json:"add_time"`                   // 创建时间
	UpdateTime   TimeNormal `gorm:"column:update_time;autoCreateTime;comment:更新时间" json:"update_time"`             // 更新时间
}

// TableName CrmProductFlowSnapshotConfig's table name
func (*CrmProductFlowSnapshotConfig) TableName() string {
	return TableNameCrmProductFlowSnapshotConfig
}
