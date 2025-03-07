package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"record-traffic-press/constant/common"
	"record-traffic-press/constant/rspcode"
	settings2 "record-traffic-press/goreplay/settings"
	"record-traffic-press/model"
	"time"
)

type RecordController struct{}

func (r RecordController) Index(context *gin.Context) {
	username, _ := context.Get("username")
	fmt.Println(username)

	//类型断言
	v, ok := username.(string)
	if ok {
		context.String(200, "用户列表--"+v)
	} else {
		context.String(200, "用户列表--获取用户失败")
	}
}

func (r RecordController) List(context *gin.Context) {
	username, _ := context.Get("username")
	fmt.Println(username)

	//类型断言
	v, ok := username.(string)
	if ok {
		context.String(200, "用户列表--"+v)
	} else {
		context.String(200, "用户列表--获取用户失败")
	}
}

func (r RecordController) Detail(context *gin.Context) {
	username, _ := context.Get("username")
	fmt.Println(username)

	//类型断言
	v, ok := username.(string)
	if ok {
		context.String(200, "用户列表--"+v)
	} else {
		context.String(200, "用户列表--获取用户失败")
	}
}

func (r RecordController) Add(context *gin.Context) {

	var settings settings2.AppSettings

	if err := context.ShouldBindJSON(&settings); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settingsJson, err := json.Marshal(settings)

	recordTraffic := model.RecordTraffic{
		BaseModel: model.BaseModel{
			Flag:       &common.NumberZero,
			CreateTime: time.Now().Unix(),
			UpdateTime: time.Now().Unix(),
			OrderId:    &common.NumberZero,
		},
		StartTime: time.Now().Unix(),
		EndTime:   time.Now().Unix(),
		Settings:  string(settingsJson),
		Status:    common.RecordStatusInit.Code,
	}

	err = model.GetRecordTrafficDAO().Insert(&recordTraffic)

	if err != nil {
		context.JSON(http.StatusOK, rspcode.Fail)
		return
	}

	context.JSON(http.StatusOK, rspcode.Success)
}

func (r RecordController) Edit(context *gin.Context) {
	username, _ := context.Get("username")
	fmt.Println(username)

	//类型断言
	v, ok := username.(string)
	if ok {
		context.String(200, "用户列表--"+v)
	} else {
		context.String(200, "用户列表--获取用户失败")
	}
}
