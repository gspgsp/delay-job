package delayjob

import (
	"delay-job/model"
	"delay-job/utils"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
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

//实时扫描ready-queue-topic队列，取出所有已经到期的job
func tickConsumerHandler() {
	//通过Blop+Ticker的话，其实有个问题，如果同一时刻有多个到期都需要处理呢，是不是必须要等到下一秒，
	//那么在这等待的一秒钟里面，订单其实已过期了,万一用户还做了操作呢。万一订单超级多，有可能要等到好几分钟以后才能处理完，而在这几分钟以内订单其实是已过期的
	//通过添加很多个ticker好像不行，默认这里有3个，相当于同一秒可以处理3个job。
	//优化的对象....
	jobId, err := blockPopFromReadyQueue(defaultTopicSlice, config.Setting.QueueBlockTimeout)
	if err != nil {
		log.Printf("the ready queue error is:%v", err.Error())
		return
	}

	if len(jobId) == 0 {
		log.Printf("未查找到相关job\n")
		return
	}

	job, err := getJob(jobId)
	if err != nil {
		log.Printf("获取job信息出错:%v\n", err.Error())
		return
	}

	updatedAt, _ := utils.FormatLocalTime(time.Now())

	//关闭会员订单操作
	if job.Topic == "close_vip_order" {
		var closeJobOrder model.CloseVipOrder
		extra := `'{"expired_reason":"订单预期未支付，系统自动取消"}'`
		if err := json.Unmarshal([]byte(job.Body), &closeJobOrder); err != nil {
			fmt.Printf("解析任务信息出错:%v\n", err.Error())
			return
		}

		sql := fmt.Sprintf("update h_vip_orders set status = -1, extra = %s, updated_at = %s where id = %d", extra, "'"+updatedAt+"'", closeJobOrder.OrderId)
		if err := baseDb.Exec(sql).Error; err != nil {
			log.Printf("更新vip订单出错:%v\n", err.Error())
			return
		}

		log.Printf("更新vip订单成功\n")
		return
	}

	//其他类型jod操作
}
