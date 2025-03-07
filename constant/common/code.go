package common

import "record-traffic-press/config/conf"

// 常量定义
var (
	DEV  = "dev"
	PROD = "prod"
	TEST = "test"
	PRE  = "pre"
)

func IsDevEnv() bool {
	return conf.GetAppConf().Env == DEV
}

func IsTestEnv() bool {
	return conf.GetAppConf().Env == TEST
}

func IsPreEnv() bool {
	return conf.GetAppConf().Env == PRE
}

func IsProdEnv() bool {
	return conf.GetAppConf().Env == PROD
}

var NumberZero int32 = 0
var NumberOne int32 = 1
var NumberTwo int32 = 2
