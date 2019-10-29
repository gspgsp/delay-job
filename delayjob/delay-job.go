package delayjob

import (
	"delay-job/model"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"delay-job/config"
)

var (
	// 每个定时器对应一个bucket
	timers []*time.Ticker
	// bucket名称chan
	bucketNameChan <-chan string
	//默认任务切片
	defaultTopicSlice = []string{"close_vip_order"}
)

// Init 初始化延时队列
func Init() {
	//从连接池获取redis
	RedisPool = initRedisPool()

	//轮询获取bucket
	bucketNameChan = generateBucketName()

	//初始化时间相关业务
	initTimers()

	//通过goroutine实时消费ready-queue
	consumerTopic()
}

// Push 添加一个Job到队列中
func Push(job Job) error {
	if job.Id == "" || job.Topic == "" || job.Delay < 0 || job.TTR <= 0 {
		return errors.New("invalid job")
	}

	err := putJob(job.Id, job)
	if err != nil {
		log.Printf("添加job到job pool失败#job-%+v#%s", job, err.Error())
		return err
	}
	err = pushToBucket(<-bucketNameChan, job.Delay, job.Id)
	if err != nil {
		log.Printf("添加job到bucket失败#job-%+v#%s", job, err.Error())
		return err
	}

	return nil
}

// Pop 轮询获取Job
func Pop(topics []string) (*Job, error) {
	jobId, err := blockPopFromReadyQueue(topics, config.Setting.QueueBlockTimeout)
	if err != nil {
		return nil, err
	}

	// 队列为空
	if jobId == "" {
		return nil, nil
	}

	// 获取job元信息
	job, err := getJob(jobId)
	if err != nil {
		return job, err
	}

	// 消息不存在, 可能已被删除
	if job == nil {
		return nil, nil
	}

	//timestamp := time.Now().Unix() + job.TTR
	//err = pushToBucket(<-bucketNameChan, timestamp, job.Id)

	return job, err
}

// Remove 删除Job
func Remove(jobId string) error {
	return removeJob(jobId)
}

// Get 查询Job
func Get(jobId string) (*Job, error) {
	job, err := getJob(jobId)
	if err != nil {
		return job, err
	}

	// 消息不存在, 可能已被删除
	if job == nil {
		return nil, nil
	}
	return job, err
}

// 轮询获取bucket名称, 使job分布到不同bucket中, 提高扫描速度
func generateBucketName() <-chan string {
	c := make(chan string)
	go func() {
		i := 1
		for {
			c <- fmt.Sprintf(config.Setting.BucketName, i)
			if i >= config.Setting.BucketSize {
				i = 1
			} else {
				i++
			}
		}
	}()

	return c
}

// 初始化定时器
func initTimers() {
	timers = make([]*time.Ticker, config.Setting.BucketSize)
	var bucketName string
	for i := 0; i < config.Setting.BucketSize; i++ {
		timers[i] = time.NewTicker(1 * time.Second)
		bucketName = fmt.Sprintf(config.Setting.BucketName, i+1)
		go waitTicker(timers[i], bucketName)
	}
}

func waitTicker(timer *time.Ticker, bucketName string) {
	for {
		select {
		case t := <-timer.C:
			tickHandler(t, bucketName)
		}
	}
}

// 扫描bucket, 取出延迟时间小于当前时间的Job
func tickHandler(t time.Time, bucketName string) {
	for {
		bucketItem, err := getFromBucket(bucketName)
		if err != nil {
			log.Printf("扫描bucket错误#bucket-%s#%s", bucketName, err.Error())
			return
		}

		// 集合为空
		if bucketItem == nil {
			return
		}

		// 延迟时间未到
		if bucketItem.timestamp > t.Unix() {
			return
		}

		// 延迟时间小于等于当前时间, 取出Job元信息并放入ready queue
		job, err := getJob(bucketItem.jobId)
		if err != nil {
			log.Printf("获取Job元信息失败#bucket-%s#%s", bucketName, err.Error())
			continue
		}

		// job元信息不存在, 从bucket中删除
		if job == nil {
			removeFromBucket(bucketName, bucketItem.jobId)
			continue
		}

		// 再次确认元信息中delay是否小于等于当前时间
		if job.Delay > t.Unix() {
			// 从bucket中删除旧的jobId
			removeFromBucket(bucketName, bucketItem.jobId)
			// 重新计算delay时间并放入bucket中
			pushToBucket(<-bucketNameChan, job.Delay, bucketItem.jobId)
			continue
		}

		err = pushToReadyQueue(job.Topic, bucketItem.jobId)
		if err != nil {
			log.Printf("JobId放入ready queue失败#bucket-%s#job-%+v#%s",
				bucketName, job, err.Error())
			continue
		}

		// 从bucket中删除
		removeFromBucket(bucketName, bucketItem.jobId)
	}
}

/**
每隔1秒就去ready_queue拉取job
*/
func consumerTopic() {
	timers = make([]*time.Ticker, config.Setting.BucketSize)
	for i := 0; i < config.Setting.BucketSize; i++ {
		timers[i] = time.NewTicker(1 * time.Second)
		go waitConsumerTicker(timers[i])
	}
}

func waitConsumerTicker(timer *time.Ticker) {
	for {
		select {
		case <-timer.C:
			tickConsumerHandler()
		}
	}
}

func tickConsumerHandler() {
	jobId, err := blockPopFromReadyQueue(defaultTopicSlice, config.Setting.QueueBlockTimeout)
	if err != nil {
		//fmt.Printf("the ready queue error is:%v", err.Error())
		return
	}

	if len(jobId) == 0 {
		//fmt.Printf("未查找到相关job\n")
		return
	}

	job, err := getJob(jobId)
	if err != nil {
		//fmt.Printf("获取job信息出错:%v\n", err.Error())
		return
	}

	//关闭会员订单操作
	if job.Topic == "close_vip_order" {
		var closeJobOrder model.CloseVipOrder
		extra := `'{"expired_reason":"订单预期未支付，系统自动取消"}'`
		if err := json.Unmarshal([]byte(job.Body), &closeJobOrder); err != nil {
			fmt.Printf("解析任务信息出错:%v\n", err.Error())
			return
		}

		sql := fmt.Sprintf("update h_vip_orders set status = -1, extra = %s where id = %d", extra, closeJobOrder.OrderId)
		if err := baseDb.Exec(sql).Error; err != nil {
			fmt.Printf("更新vip订单出错:%v\n", err.Error())
			return
		}

		fmt.Printf("更新vip订单成功\n")
		return
	}

	//其他操作
}
