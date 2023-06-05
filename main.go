package main

import (
	conf "chatAi/config"
)

func main() {
	conf.Config()
	conf.GinServer.Run(":9969")
}
