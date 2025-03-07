package rspcode

import "fmt"

// RspCode 错误码结构体
type RspCode struct {
	Code int32  // 错误码
	Msg  string // 错误信息
}

// 实现 error 接口
func (e *RspCode) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Code, e.Msg)
}

var (
	Success = &RspCode{Code: 200, Msg: "OK"}
	Fail    = &RspCode{Code: 500, Msg: "操作失败"}

	InvalidParameter       = &RspCode{Code: 400, Msg: "无效参数"}
	InvalidParameterLawful = &RspCode{Code: 400, Msg: "参数不合法"}
	RequiredLogin          = &RspCode{Code: 401, Msg: "需要登录"}
	InvalidSignature       = &RspCode{Code: 402, Msg: "无效签名"}
	NoPermission           = &RspCode{Code: 403, Msg: "没有权限"}
	NotExist               = &RspCode{Code: 404, Msg: "不存在的数据"}
	SystemError            = &RspCode{Code: 501, Msg: "系统错误"}
	Expire                 = &RspCode{Code: 504, Msg: "超时"}
	DataWrong              = &RspCode{Code: 507, Msg: "数据不正确"}
)
