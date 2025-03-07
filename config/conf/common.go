package conf

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

const (
	confPath = "config/conf/conf.yaml"
)

type (
	// AppConf 应用配置
	AppConf struct {
		ProjectName string      `yaml:"project_name"`
		DBConfList  []DBConf    `yaml:"db" validate:"required,dive"`
		RedisConfig RedisConfig `yaml:"redis"`
		Env         string      `yaml:"env"`
	}

	// DBConf 数据库配置文件
	DBConf struct {
		Name     string `yaml:"name"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"db_name"`
		Type     string `yaml:"type"`
	}

	// RedisConfig redis配置
	RedisConfig struct {
		Host        string `yaml:"host"`
		Port        int    `yaml:"port"`
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		DB          int    `yaml:"db"`
		MaxPoolSize int    `yaml:"max_pool_size"`
		MinPoolSize int    `yaml:"min_pool_size"`
		CacheEnable bool   `yaml:"cache_enable"`
		Fix         string `yaml:"fix"`
	}
)

var (
	conf     AppConf
	validate = validator.New()
)

// GetAppConf 获取应用配置信息
func GetAppConf() *AppConf {
	return &conf
}

// ReadFromLocal 从本地读取配置文件
func ReadFromLocal() {

	fBytes, err := ioutil.ReadFile(confPath)

	if err != nil {
		panic(fmt.Sprintf("读取本地配置文件遇到错误:%s,请检查是否存在本地配置文件,路径为:%s", err.Error(), confPath))
	}

	if err = yaml.Unmarshal(fBytes, &conf); err != nil {
		panic(fmt.Sprintf("parsing the configuration file failed:%s", err.Error()))
	}

	if err = validate.Struct(&conf); err != nil {
		panic(fmt.Sprintf("checking the configuration file failed,%s", err.Error()))
	}
}

// String 将数据库配置拼接为string
func (t DBConf) String() string {
	return fmt.Sprintf("[DB info] addr:%s:%s@%s:%d db:%s", t.User, t.Password, t.Host, t.Port, t.DBName)
}
