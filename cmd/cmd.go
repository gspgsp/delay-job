package cmd

import (
	"delay-job/config"
	"delay-job/delayjob"
	"delay-job/logs"
	"delay-job/routers"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

// Cmd 应用入口Command
type Cmd struct{}

var (
	version    bool
	configFile string
)

const (
	// AppVersion 应用版本号
	AppVersion = "0.1"
)

// Run 运行应用
func (cmd *Cmd) Run() {

	// 解析命令行参数
	cmd.parseCommandArgs()
	if version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	//初始化日志配置
	logs.InitFilesystemLogger()

	// 初始化配置
	config.Init(configFile)

	//初始化mysql数据库配置
	new(delayjob.BaseOrm).InitDB()

	// 初始化队列
	delayjob.Init()

	// 运行web server
	cmd.runWeb()
}

// 解析命令行参数
func (cmd *Cmd) parseCommandArgs() {
	// 配置文件
	flag.StringVar(&configFile, "c", "", "./delay-job -c /path/to/delay-job.conf")
	// 版本
	flag.BoolVar(&version, "v", false, "./delay-job -v")
	flag.Parse()
}

// 运行Web Server
func (cmd *Cmd) runWeb() {
	http.HandleFunc("/close_vip_order", routers.CloseVipOrder)

	log.Printf("listen %s\n", config.Setting.BindAddress)
	err := http.ListenAndServe(config.Setting.BindAddress, nil)
	log.Fatalln(err)
}
