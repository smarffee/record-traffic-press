package proto

import hessian "github.com/apache/dubbo-go-hessian2"

// DubboHeader Dubbo 协议头
type DubboHeader struct {
	hessian.POJO
	Magic      [2]byte // 魔数
	Flag       byte    // 标志位
	Status     byte    // 状态
	RequestID  int64   // 请求 ID
	DataLength int32   // 数据长度
}

// DubboBody Dubbo 请求体
type DubboBody struct {
	DubboVersion   string        // Dubbo 版本
	ServiceName    string        // 服务接口名
	ServiceVersion string        // 服务版本
	MethodName     string        // 方法名
	ParameterTypes []string      // 参数类型
	Arguments      []interface{} // 参数值
}
