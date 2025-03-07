package routers

import (
	"github.com/gin-gonic/gin"
	"record-traffic-press/controller"
	"record-traffic-press/middlewares"
)

func RecordControllerRoutersInit(r *gin.Engine) {
	//middlewares.InitMiddleware中间件
	recordRouters := r.Group("/record", middlewares.InitMiddleware)
	{
		recordRouters.GET("/", controller.RecordController{}.Index)

		recordRouters.GET("/list", controller.RecordController{}.List)
		recordRouters.GET("/detail", controller.RecordController{}.Detail)
		recordRouters.POST("/add", controller.RecordController{}.Add)
		recordRouters.POST("/edit", controller.RecordController{}.Edit)
	}
}
