package main

import (
	"github.com/sirupsen/logrus"
	"record-traffic-press/config/conf"
	"record-traffic-press/config/db"
	"record-traffic-press/model"
	"record-traffic-press/routers"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func main() {

	// 设置日志格式为 JSON
	//logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetFormatter(&logrus.TextFormatter{})

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)

	// 初始化配置文件
	conf.ReadFromLocal()

	// 初始化数据库
	db.InitialDB()

	// 初始化DAO
	model.InitialTable()

	// 创建一个默认的路由引擎
	r := gin.Default()

	// 自定义模板函数  注意要把这个函数放在加载模板前
	//r.SetFuncMap(template.FuncMap{
	//	"UnixToTime": models.UnixToTime,
	//})

	// 加载模板 放在配置路由前面
	//r.LoadHTMLGlob("templates/**/*")
	// 配置静态web目录   第一个参数表示路由, 第二个参数表示映射的目录
	//r.Static("/static", "./static")

	// 配置session中间件
	store, _ := redis.NewStore(10, "tcp", "localhost:6379", "", []byte("secret111"))
	r.Use(sessions.Sessions("mysession", store))

	routers.RecordControllerRoutersInit(r)

	r.Run()
}
