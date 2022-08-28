package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/zooyer/embed"
	"github.com/zooyer/embed/log"
	"github.com/zooyer/sshd/common/conf"
)

var (
	passwd = flag.String("passwd", "", "hash password")
	config = flag.String("config", "conf/conf.test.yaml", "config file")
)

func init() {
	flag.Parse()

	if *passwd != "" {
		fmt.Println("password:", hashPassword(*passwd))
		os.Exit(0)
	}

	initEmbed()
	initConfig()
	initLog()
}

func initEmbed() {
	embed.Init()
}

func initConfig() {
	if err := conf.Init(*config); err != nil {
		panic(err)
	}
}

func initLog() {
	log.Init(&conf.Log)
}
