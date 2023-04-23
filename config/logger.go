package config

import (
	"io"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func init() {
	// 设置日志格式为json格式
	log.SetFormatter(&log.JSONFormatter{})

	// 设置将日志输出到标准输出（默认的输出为stderr，标准错误）
	// 日志消息输出可以是任意的io.writer类型
	logFile, err := os.OpenFile("./ai_server_logger.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic("创建日志文件失败")
	}
	logLocal := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(logLocal)
	log.SetFormatter(&log.TextFormatter{})
	// 设置日志级别为debug以上
	log.SetLevel(log.DebugLevel)
}

// 日志记录到文件
func LoggerToFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()
		// 请求方式
		reqMethod := c.Request.Method
		// 请求路由
		reqUri := c.Request.RequestURI
		// 请求IP
		clientIP := c.ClientIP()
		// 日志格式
		log.Infof("| 请求ip:%15s | 请求方法:%s | 请求路径:%s |",
			clientIP,
			reqMethod,
			reqUri,
		)
		// 处理请求
		c.Next()
		// 结束时间
		endTime := time.Now()
		// 执行时间
		latencyTime := endTime.Sub(startTime)
		// 状态码
		statusCode := c.Writer.Status()
		// 日志格式
		log.Infof("| 请求ip:%15s | 请求方法:%s | 请求路径:%s | 响应状态:%3d | 耗时:%13v |",
			clientIP,
			reqMethod,
			reqUri,
			statusCode,
			latencyTime,
		)
	}
}
