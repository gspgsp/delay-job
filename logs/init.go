package logs

import (
	"delay-job/config"
	"github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"path"
	"time"
)

//切割日志和清理过期日志
func InitFilesystemLogger() {
	//格式换时间输出格式
	log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05"})

	//日志路劲
	filePath := path.Join(config.DefaultLogPath, config.DefaultLogName)

	writer, err := rotatelogs.New(
		filePath+".%Y%m%d.log",                    //%Y%m%d%H%M"，日志分割时间
		rotatelogs.WithLinkName(filePath),         // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(time.Hour*7*24),     // 文件最大保存时间
		rotatelogs.WithRotationTime(time.Hour*24), // 日志切割时间间隔
	)
	if err != nil {
		log.Fatal("Init log failed, err:", err)
	}
	log.SetOutput(writer)
}
