package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	conf "chatAi/config"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

var (
	client      *openai.Client
	openaikey   string
	flowChannel = make(chan byte, 10)
)

func init() {
	flag.StringVar(&openaikey, "key", "", "openaikey")
	flag.Parse()
	log.Info("openaikey:", openaikey)

	client = openai.NewClient(openaikey)
}

type CallRequest struct {
	Word string `json:"word"`
}

func main() {

	r := gin.Default()
	//日志
	r.Use(conf.LoggerToFile())
	//跨域
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 30 * time.Minute,
	}))
	go func() {
		tick := time.Tick(5 * time.Second)
		for {
			select {
			case <-tick:
				flowChannel <- 1
				log.Info("令牌桶已放行,当前容量:", len(flowChannel))
			}
		}
	}()
	r.POST("/call", func(c *gin.Context) {
		//请求校验
		var callRequest CallRequest
		if err := c.ShouldBind(&callRequest); err != nil {
			log.Info("入参校验不通过为:", err)
			c.JSON(http.StatusBadRequest, gin.H{"msg": "参数错误"})
			return
		}
		if size := len(callRequest.Word); size >= 1000 {
			log.Info("请求体过长:", size)
			c.JSON(http.StatusBadRequest, gin.H{"msg": "请求信息过长"})
			return
		}
		log.Info("请求入参为:", callRequest.Word)
		//调用ai
		queue := make(chan string)
		select {
		case <-flowChannel:
			log.Info("令牌桶放行,当前容量:", len(flowChannel))
			go callOpenAi(callRequest.Word, queue)
		case <-time.After(1 * time.Millisecond):
			log.Info("服务器繁忙!")
			c.JSON(http.StatusGatewayTimeout, gin.H{"msg": "服务器繁忙"})
			return
		}
		select {
		case result := <-queue:
			log.Info("Ai响应为:", result)
			c.JSON(http.StatusOK, gin.H{"msg": result})
		case <-time.After(30 * time.Second):
			log.Info("请求超时!")
			c.JSON(http.StatusGatewayTimeout, gin.H{"msg": "服务响应超时"})
		}
	})
	r.Run(":9969")

}

func callOpenAi(word string, done chan string) {
	//发送请求
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			MaxTokens: 4000,
			Model:     "gpt-3.5-turbo",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: word,
				},
			},
		},
	)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		done <- "服务繁忙请稍后再试"
		return
	}
	done <- resp.Choices[0].Message.Content
}
