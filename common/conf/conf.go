package conf

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/zooyer/embed/log"
)

var Viper = viper.New()

var (
	Addr   string
	Key    string
	Banner string
	Shell  string
	Env    []string
	Log    log.Config
	User   = make(map[string]string)
)

var unmarshal = map[string]interface{}{
	"addr":   &Addr,
	"env":    &Env,
	"log":    &Log,
	"banner": &Banner,
	"shell":  &Shell,
	"key":    &Key,
	"user":   &User,
}

func onUnmarshal() {
	for key, addr := range unmarshal {
		if err := Viper.UnmarshalKey(key, addr); err != nil {
			return
		}
	}
}

func Init(filename string) (err error) {
	Viper.SetConfigFile(filename)
	Viper.WatchConfig()
	if err = Viper.ReadInConfig(); err != nil {
		return
	}

	Viper.OnConfigChange(func(in fsnotify.Event) {
		onUnmarshal()
	})

	onUnmarshal()

	return
}
