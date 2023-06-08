package config

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var GinServer *gin.Engine = gin.Default()

func Gin() {
	//日志
	GinServer.Use(LoggerToFile())
	//跨域
	webCross()
	//session
	webSession()
}
func webCross() {
	cors.DefaultConfig()
	GinServer.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://afflatus.wang", "https://afflatus.wang"},
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "OPTIONS"},
		AllowHeaders:     []string{"*", "Content-Type"},
		ExposeHeaders:    []string{"Set-Cookie", "*"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 30 * time.Minute,
	}))
}
func webSession() {
	host := viper.GetString("server.host")
	store := memstore.NewStore([]byte("duiashdiuwdbnedusdsadwda2"))
	GinServer.Use(sessions.Sessions("afflatus-session", store))
	GinServer.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Options(sessions.Options{MaxAge: 1800, Path: "/", Domain: host, Secure: false, HttpOnly: true})
		c.Set("threadLocalSession", session)
		c.Next()
	})

}
