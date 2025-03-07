package model

import "github.com/sirupsen/logrus"

// TableName 表名
func (s *RecordTraffic) TableName() string {
	return "record_traffic"
}

type RecordTraffic struct {
	BaseModel
	StartTime int64  `gorm:"column:start_time;type:TIMESTAMP;comment:'开始时间'" json:"start_time"`
	EndTime   int64  `gorm:"column:end_time;type:TIMESTAMP;comment:'结束时间'" json:"end_time"`
	Settings  string `gorm:"column:settings;type:varchar(102);comment:'配置信息'" json:"settings"`
	Status    int32  `gorm:"column:status;type:int;comment:'状态, 1:初始化; 2:进行中; 3:结束;'" json:"status"`
}

// RecordTrafficDAO 数据库访问对象
type RecordTrafficDAO struct {
	BaseDAO
}

var recordTrafficDAO RecordTrafficDAO

func GetRecordTrafficDAO() *RecordTrafficDAO {
	return &recordTrafficDAO
}

// Insert 保存
func (t *RecordTrafficDAO) Insert(recordTraffic *RecordTraffic) error {
	// 插入后返回主键
	err := t.db.Create(recordTraffic).Error

	if err != nil {
		logrus.Errorf("insert study course pack failed. err:%v", err)
		return err
	}

	return nil
}
