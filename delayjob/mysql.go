package delayjob

import (
	"delay-job/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"log"
	"time"
)

var baseDb *gorm.DB

type BaseOrm struct {
	DB *gorm.DB
}

/**
初始化数据库
*/
func (that *BaseOrm) InitDB() {
	var err error

	that.DB, err = gorm.Open(config.Setting.Mysql.DbConnect, config.Setting.Mysql.DbUserName+":"+config.Setting.Mysql.DbPassword+"@tcp("+config.Setting.Mysql.DbHost+":"+config.Setting.Mysql.DbPort+")/"+config.Setting.Mysql.DbDatabase+"?charset=utf8&parseTime=true&loc=Local")

	if err != nil {
		log.Println("init db error!", err)
		return
	}

	that.DB.SingularTable(true)
	that.DB.DB().SetMaxIdleConns(10)
	that.DB.DB().SetMaxOpenConns(100)
	that.DB.DB().SetConnMaxLifetime(300 * time.Second)
	that.DB.LogMode(true)

	baseDb = that.DB
}
