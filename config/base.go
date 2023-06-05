package config

import (
	router "chatAi/router"
	"time"
)

// 配置初始化
func Config() {
	//设置时区
	timeLocal()
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
