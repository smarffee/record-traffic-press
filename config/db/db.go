// Package db 连接库
package db

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"record-traffic-press/config/conf"
	"time"
)

// 不同数据源名称标识
const (
	Traffic    = "traffic"
	TrainingDB = "training"
)

// 数据源属性配置
const (
	maxIdleConn = 2
	maxOpenConn = 20
	Mysql       = "mysql"
	Sqlserver   = "sqlserver"
)

// 数据源实例
var (
	dbMgr = map[string]*gorm.DB{}
)

// InitialDB 初始化数据库连接池
func InitialDB() {

	for _, dbConf := range conf.GetAppConf().DBConfList {
		switch dbConf.Type {
		case Mysql:
			initMySQLDB(dbConf)
		default:
			log.Fatalf("db type %s not support", dbConf.Type)
		}
	}

	logrus.Info("init db connector success, size:%d", len(conf.GetAppConf().DBConfList))
}

// GetDBBySourceName 通过数据源标识符获取db
func GetDBBySourceName(name string) (*gorm.DB, error) {
	db, ok := dbMgr[name]

	if !ok {
		return nil, errors.New("数据源不存在")
	}

	return db, nil
}

// initMySQLDB 连接MySQL数据库
func initMySQLDB(dbConf conf.DBConf) {
	// 1.设置mysql数据库连接参数
	dsn := fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local",
		dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.DBName)

	// 设置日志级别为 Info
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的地方，这里是标准输出）
		logger.Config{
			LogLevel:                  logger.Info,
			SlowThreshold:             200 * time.Millisecond, // 慢查询阈值
			IgnoreRecordNotFoundError: true,                   // 忽略未找到记录的错误
			Colorful:                  true,                   // 是否使用颜色
		},
	)

	// 2.打开mysql数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:                 newLogger,
		SkipDefaultTransaction: true,
	})

	if err != nil {
		log.Fatalf("connect db %s, err %s", dbConf, err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(maxIdleConn)
	sqlDB.SetMaxOpenConns(maxOpenConn)

	dbMgr[dbConf.Name] = db
}
