// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameCrmProductFlowSnapshotDetail = "crm_product_flow_snapshot_detail"

// CrmProductFlowSnapshotDetail 产品流水月度快照明细表
type CrmProductFlowSnapshotDetail struct {
	ID                    int32     `gorm:"column:id;primaryKey;autoIncrement:true;comment:主键id" json:"id"`                        // 主键id
	Month 				  string    `gorm:"column:month;comment:月份" json:"month"` 													 // 产品月度流水快照id
	MediaID               int32     `gorm:"column:media_id;comment:媒体平台" json:"media_id"`                                      // 产品id
	ProductID             int32     `gorm:"column:product_id;comment:产品id" json:"product_id"`                                      // 产品id
	AdvertiserID          string    `gorm:"column:advertiser_id;comment:媒体账号id" json:"advertiser_id"`                             // 媒体账号id
	SnapShotURL           string    `gorm:"column:snap_shot_url;comment:截图url" json:"snap_shot_url"`                               // 截图url
	AddTime               TimeNormal `gorm:"column:add_time;autoCreateTime;comment:创建时间" json:"add_time"`                                        // 创建时间
	UpdateTime            TimeNormal `gorm:"column:update_time;autoCreateTime;comment:更新时间" json:"update_time"`                                  // 更新时间
}

// TableName CrmProductFlowSnapshotDetail's table name
func (*CrmProductFlowSnapshotDetail) TableName() string {
	return TableNameCrmProductFlowSnapshotDetail
}