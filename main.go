package main

import (
	conf "chatAi/config"
	"strconv"

	viper "github.com/spf13/viper"
)

func main() {
	conf.Config()
	port := viper.GetInt64("server.port")
	conf.GinServer.Run(":" + strconv.FormatInt(port, 10))
}
