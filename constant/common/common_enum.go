package common

// EnumType 课单类型
type EnumType struct {
	Code int32  // 枚举值
	Desc string // 课单类型
}

// InvalidZeroValue 无效值
var InvalidZeroValue = EnumType{Code: 0, Desc: "无效值"}

// YesOrNo 是或者否类型枚举
var (
	No  = EnumType{Code: 0, Desc: "否"}
	Yes = EnumType{Code: 1, Desc: "是"}
)

// 课单评价审批状态枚举
var (
	RecordStatusInit      = EnumType{Code: 1, Desc: "初始化"}
	RecordStatusRecording = EnumType{Code: 2, Desc: "进行中"}
	RecordStatusFinished  = EnumType{Code: 3, Desc: "结束"}
)
