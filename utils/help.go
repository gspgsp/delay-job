package utils

import (
	"delay-job/model"
	"errors"
	"strconv"
	"time"
)

/**
格式化时间为local时间
*/
func FormatLocalTime(time time.Time) (str string, err error) {
	jsonTime := model.JsonTime(time)
	if str := strconv.Quote((&jsonTime).String()); len(str) > 0 {
		//去掉引号
		return strconv.Unquote(str)
	}

	return "", errors.New("解析错误")
}
