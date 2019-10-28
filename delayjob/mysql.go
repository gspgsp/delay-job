package delayjob

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

type BaseOrm struct {
	DB *gorm.DB
}
