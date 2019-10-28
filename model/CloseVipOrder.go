package model

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
