package model

import (
	"delay-job/config"
	"fmt"
	"time"
)

/**
会员订单job模型
*/
type CloseVipOrder struct {
	OrderId int64 `json:"order_id"`
}

/**
拓展字段
*/
type Extra struct {
	ExpiredReason string `json:"expired_reason"`
}

/**
时间格式化
*/
type JsonTime time.Time

/**
返回格式化时间字符串
*/
func (this *JsonTime) String() string {
	return fmt.Sprintf("%s", time.Time(*this).Format(config.DefaultTimeFormat))
}
