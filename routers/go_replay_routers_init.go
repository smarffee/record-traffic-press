package routers

import (
	"github.com/buger/controller/goreplay"
	"github.com/gin-gonic/gin"
)

func GoReplayRoutersInit(r *gin.Engine) {

	goreplayRouters := r.Group("/goreplay")
	{
		goreplayRouters.POST("/start", goreplay.GoReplayController{}.Start)
	}

}
