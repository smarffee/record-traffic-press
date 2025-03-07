package model

import (
	"gorm.io/gorm"
	"record-traffic-press/config/db"
)

// 配置每个DAO使用的数据源
var tableList = []TableWrapper{
	// 使用 Traffic 数据源
	{GetRecordTrafficDAO(), db.Traffic},
}

type (
	// BaseDAO DAO对象公有组件
	BaseDAO struct {
		dbName string
		db     *gorm.DB
	}

	// DBSetter 关联数据库源, 所有DAO对象都实现了此接口
	DBSetter interface {
		setDB(dbName string, db *gorm.DB)
	}

	TableWrapper struct {
		dao    DBSetter
		dbName string
	}
)

// InitialTable 初始化表数据,需要在数据库初始化完成后执行
func InitialTable() {
	// tableList 注册使用到的表
	for _, table := range tableList {
		dbIns, err := db.GetDBBySourceName(table.dbName)
		if err != nil {
			panic(err.Error())
		}
		table.dao.setDB(table.dbName, dbIns)
	}
}

// DBName select from db
func (t *BaseDAO) setDB(dbName string, db *gorm.DB) {
	t.dbName, t.db = dbName, db
}
