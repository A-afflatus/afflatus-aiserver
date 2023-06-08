package config

import (
	"bytes"
	"flag"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	Model *string = flag.String("runProfile", "", "runProfile")
)

func ConfigInit() {
	log.Info("runProfile:", *Model)
	fileByte, err := ioutil.ReadFile("conf/" + *Model + ".json")
	if err != nil {
		log.Error("读取配置文件失败", err)
	}
	// 整合配置文件 使用viper
	viper.SetConfigType("json")
	e := viper.ReadConfig(bytes.NewBuffer(fileByte))
	if e != nil {
		log.Error("读取配置文件失败", e)
		panic(e)
	}
}
