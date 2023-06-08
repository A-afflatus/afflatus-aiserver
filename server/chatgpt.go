package server

import (
	"context"
	"flag"
	"strings"
	"time"

	"encoding/gob"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

func init() {
	//向gob注册序列话类型
	gob.Register([]openai.ChatCompletionMessage{})
}

const (
	RecordsSeessionKey = "chatting_records"
)

type Worker struct {
	workerId int
	client   *openai.Client
}
type CallRequest struct {
	Word string `json:"word"`
}

var (
	clientPool    []Worker
	openaikey     *string = flag.String("key", "", "openaikey")
	tokenCapacity         = 10
	poolCount             = 5
	flowChannel           = make(chan byte, tokenCapacity)
	workerChannel         = make(chan Worker, poolCount)
)

// 初始化ai服务
func aiserver() {
	log.Info("openaikey:", *openaikey)
	//池子容量
	for i := 0; i < poolCount; i++ {
		clientPool = append(clientPool, Worker{workerId: i, client: openai.NewClient(*openaikey)})
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
	//初始化令牌桶
	go func() {
		for i := 0; i < tokenCapacity; i++ {
			flowChannel <- 1
		}
		tick := time.Tick(10 * time.Second)
		for {
			<-tick
			flowChannel <- 1
		}
	}()
}

// 调用ai接口
func callOpenAi(session sessions.Session, word string, done chan<- string) {
	worker := <-workerChannel
	var reqMessage []openai.ChatCompletionMessage
	records := session.Get(RecordsSeessionKey)
	if records == nil {
		reqMessage = []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "用markdown格式回复代码块指定代码语言类型",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: word,
			},
		}
	} else {
		reqMessage = records.([]openai.ChatCompletionMessage)
		reqMessage = append(reqMessage, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: word,
		})
	}
	resp, err := worker.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo0301,
			Messages: reqMessage,
		},
	)
	if err != nil {
		log.Error("openai调用失败,响应信息为", err)
		if strings.Contains(err.Error(), "This model's maximum context length") {
			log.Warn("当前问答已到达最大token限制,重制session中对话记录")
			session.Delete(RecordsSeessionKey)
			done <- "<font color='red'><b>上下文已超过最大限制,下文的对话将重新计算上下文!</b></font>"
			return
		}
		if strings.Contains(err.Error(), "Rate limit reached") {
			log.Warn("当前问答频率过高,限制一分钟内最多支持3次问答")
			done <- "<font color='red'><b>您的请求太频繁!一分钟内最多支持3次问答</b></font>"
			return
		}
		done <- "AI服务繁忙请稍后再试!"
		// done <- "站主余额不足,待充值后开放!"
		return
	}
	doneStr := resp.Choices[0].Message.Content
	reqMessage = append(reqMessage, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: doneStr,
	})
	session.Set(RecordsSeessionKey, reqMessage)
	session.Save()
	done <- doneStr
}

// 注册http接口
func CallHttp(c *gin.Context) {
	session := c.MustGet("threadLocalSession").(sessions.Session)
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
		//调用ai
		go callOpenAi(session, callRequest.Word, queue)
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
}

type CallRecord struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 注册http接口
func CallRecordHttp(c *gin.Context) {
	session := c.MustGet("threadLocalSession").(sessions.Session)
	records := session.Get(RecordsSeessionKey)
	result := []CallRecord{}
	if records != nil {
		list := records.([]openai.ChatCompletionMessage)
		for _, v := range list {
			if v.Role != openai.ChatMessageRoleSystem {
				result = append(result, CallRecord{Role: v.Role, Content: v.Content})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"result": result})
}
