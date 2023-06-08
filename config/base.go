package config

import (
	router "chatAi/router"
	"flag"
	"time"
)

// 配置初始化
func Config() {
	flag.Parse()
	//设置时区
	timeLocal()
	//初始化配置
	ConfigInit()
	//初始化日志
	Logger()
	//初始化gin
	Gin()
	//初始化openai
	router.Router(GinServer)
}
func timeLocal() {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	time.Local = location
}
