package model

// BaseModel 通用模型
type BaseModel struct {
	ID         int32  `gorm:"column:id;type:int;primaryKey;autoIncrement;comment:'主键自增ID'" json:"id"`
	Flag       *int32 `gorm:"column:flag;type:int;default:0;comment:'是否删除(0:否,1:是)'" json:"flag"`
	CreateTime int64  `gorm:"column:create_time;type:TIMESTAMP;comment:'创建时间'" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;type:TIMESTAMP;comment:'更新时间'" json:"update_time"`
	OrderId    *int32 `gorm:"column:order_id;type:int;comment:'排序ID'" json:"order_id"`
}
