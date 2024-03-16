// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model



const TableNameCrmProductFlowSnapshot = "crm_product_flow_snapshot"

// CrmProductFlowSnapshot 产品流水月度快照
type CrmProductFlowSnapshot struct {
	ID                          int32     `gorm:"column:id;primaryKey;comment:主键id" json:"id"`                                                      // 主键id
	ProductFlowSnapshotConfigID int32     `gorm:"column:product_flow_snapshot_config_id;comment:产品流水截图配置id" json:"product_flow_snapshot_config_id"` // 产品流水截图配置id
	ProductID                   int32     `gorm:"column:product_id;comment:产品id" json:"product_id"`                                                 // 产品id
	Month                       string     `gorm:"column:month;comment:月份" json:"month"`                                                             // 月份
	ZipURL                      string    `gorm:"column:zip_url;comment:压缩包地址" json:"zip_url"`                                                      // 压缩包地址
	AddTime                     TimeNormal `gorm:"column:add_time;comment:创建时间" json:"add_time"`                                                     // 创建时间
	UpdateTime                  TimeNormal `gorm:"column:update_time;comment:更新时间" json:"update_time"`                                               // 更新时间
}

// TableName CrmProductFlowSnapshot's table name
func (*CrmProductFlowSnapshot) TableName() string {
	return TableNameCrmProductFlowSnapshot
}
