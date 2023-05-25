package main

import (
	"context"
	"flag"
	"sync"
	"time"

	conf "chatAi/config"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type Worker struct {
	workerId int
	client   *openai.Client
	M        sync.Mutex
}

var (
	//todo 链接池化
	clientPool    []Worker
	openaikey     string
	tokenCapacity = 10
	poolCount     = 5
	flowChannel   = make(chan byte, tokenCapacity)

	workerChannel = make(chan Worker, poolCount)
)

func init() {
	flag.StringVar(&openaikey, "key", "", "openaikey")
	flag.Parse()
	log.Info("openaikey:", openaikey)
	//池子容量
	for i := 0; i < poolCount; i++ {
		clientPool = append(clientPool, Worker{workerId: i, client: openai.NewClient(openaikey)})
		log.Info("初始化openai客户端池ID:", i)
	}
	//循环填充池
	go func() {
		for {
			for _, v := range clientPool {
				workerChannel <- v
			}
		}
	}()
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
		for i := 0; i < tokenCapacity; i++ {
			flowChannel <- 1
		}
		tick := time.Tick(10 * time.Second)
		for {
			select {
			case <-tick:
				flowChannel <- 1
			}
		}
	}()
	r.POST("/call", func(c *gin.Context) {
		//请求校验
		var callRequest CallRequest
		if err := c.ShouldBind(&callRequest); err != nil {
			log.Info("入参校验不通过为:", err)
			c.JSON(http.StatusBadRequest, gin.H{"msg": "参数错误!"})
			return
		}
		if size := len(callRequest.Word); size >= 400 {
			log.Info("请求体过长:", size)
			c.JSON(http.StatusBadRequest, gin.H{"msg": "请求信息过长!"})
			return
		}
		log.Info("请求入参为:", callRequest.Word)
		//调用ai
		queue := make(chan string)
		select {
		case <-flowChannel:
			log.Info("令牌桶放行,当前容量:", len(flowChannel))
			go callOpenAi(callRequest.Word+"(用markdown回复,如果内容有代码在代码块表达式中指定代码的语言类型)", queue)
		case <-time.After(1 * time.Millisecond):
			log.Info("服务器繁忙!")
			c.JSON(http.StatusGatewayTimeout, gin.H{"msg": "服务器繁忙!"})
			return
		}
		select {
		case result := <-queue:
			log.Info("Ai响应为:", result)
			c.JSON(http.StatusOK, gin.H{"msg": result})
		case <-time.After(30 * time.Second):
			log.Info("请求超时!")
			c.JSON(http.StatusGatewayTimeout, gin.H{"msg": "服务响应超时!"})
		}
	})
	r.Run(":9969")

}

func callOpenAi(word string, done chan<- string) {
	worker := <-workerChannel
	if !worker.M.TryLock() {
		done <- "连接池繁忙请稍后再试!"
	}
	log.Infof("当前使用的客户端ID为【%d】", worker.workerId)
	resp, err := worker.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			MaxTokens: 4000,
			Model:     openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "user",
					Content: word,
				},
			},
		},
	)
	worker.M.Unlock()
	if err != nil {
		log.Error("openai调用失败,响应信息为%v", err)
		done <- "AI服务繁忙请稍后再试!"
		// done <- "站主余额不足,待充值后开放!"
		return
	}
	done <- resp.Choices[0].Message.Content

}
