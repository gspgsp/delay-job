package delayjob

import (
	"errors"
	"github.com/vmihailenco/msgpack"
)

// Job 使用msgpack序列化后保存到Redis,减少内存占用
//type Job struct {
//	Topic string `json:"topic" msgpack:"1"`
//	Id    string `json:"id" msgpack:"2"`    // job唯一标识ID
//	Delay int64  `json:"delay" msgpack:"3"` // 延迟时间, unix时间戳
//	TTR   int64  `json:"ttr" msgpack:"4"`
//	Body  string `json:"body" msgpack:"5"`
//}

/**
Job体
*/
type Jobs struct {
	Topic string `json:"topic"`
	ID    string `json:"id"`
	Delay int64  `json:"delay"`
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
func getJob(key string) (*CloseOrder, error) {
	value, err := execRedisCommand("GET", key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("未找到相关job信息")
	}

	byteValue := value.([]byte)
	job := &CloseOrder{}
	err = msgpack.Unmarshal(byteValue, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// 添加Job
func putJob(key string, job CloseOrder) error {
	value, err := msgpack.Marshal(job)
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
