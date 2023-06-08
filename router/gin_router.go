package router

import (
	server "chatAi/server"

	"github.com/gin-gonic/gin"
)

// 注册路由
func Router(rootGroup *gin.Engine) {
	//先初始化服务
	server.ServerInit()
	//后开启路由
	openai(rootGroup)
}

// openai_router
func openai(rootGroup *gin.Engine) {
	openaiGroup := rootGroup.Group("/openai")
	openaiGroup.POST("/call", server.CallHttp)
	openaiGroup.GET("/callRecord", server.CallRecordHttp)
}
