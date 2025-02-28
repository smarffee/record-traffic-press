package main

import (
	"github.com/buger/routers"
	"github.com/buger/utils"
	"html/template"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	// 创建一个默认的路由引擎
	r := gin.Default()

	// 自定义模板函数  注意要把这个函数放在加载模板前
	r.SetFuncMap(template.FuncMap{
		"UnixToTime": utils.UnixToTime,
	})

	// 加载模板 放在配置路由前面
	//r.LoadHTMLGlob("templates/**/*")

	// 配置静态web目录   第一个参数表示路由, 第二个参数表示映射的目录
	r.Static("/static", "./static")

	// 配置session中间件
	store, _ := redis.NewStore(10, "tcp", "localhost:6379", "", []byte("secret111"))
	r.Use(sessions.Sessions("user-session", store))

	// 初始化流量录制controller
	routers.GoReplayRoutersInit(r)

	// 启动 HTTP 服务器
	r.Run(":9001")
}
