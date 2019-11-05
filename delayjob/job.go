package delayjob

import (
	"errors"
	"github.com/vmihailenco/msgpack"
)

/**
Job体
*/
type Jobs struct {
	Topic string `json:"topic"`
	ID    string `json:"id"`    //job唯一标识ID
	Delay int64  `json:"delay"` //延迟时间, unix时间戳
	TTR   int64  `json:"ttr"`
}

/**
关闭vip订单
*/
type CloseOrder struct {
	Jobs
	Body CloseOrderBody `json:"body"`
}

/**
关闭vip订单具体操作
*/
type CloseOrderBody struct {
	OrderId string `json:"order_id"`
}

// 获取Job
func getJob(key, topic string) (interface{}, error) {
	value, err := execRedisCommand("GET", key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("未找到相关job信息")
	}

	byteValue := value.([]byte)

	switch topic {
	case "close_vip_order":
		job := CloseOrder{}
		err = msgpack.Unmarshal(byteValue, &job)
		if err != nil {
			return nil, err
		}

		return job, nil
	default:
		return nil, nil
	}
}

// 添加Job
func putJob(key string, v interface{}) error {
	value, err := msgpack.Marshal(v)
	if err != nil {
		return err
	}
	_, err = execRedisCommand("SET", key, value)

	return err
}

// 删除Job
func removeJob(key string) error {
	_, err := execRedisCommand("DEL", key)

	return err
}
