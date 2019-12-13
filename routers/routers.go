package routers

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"delay-job/delayjob"
)

// 关闭订单Handler
func CloseOrder(resp http.ResponseWriter, req *http.Request) {
	var closeOrder delayjob.CloseOrder
	err := readBody(resp, req, &closeOrder)
	if err != nil {
		return
	}

	//new
	closeOrder.ID = strings.TrimSpace(closeOrder.ID)
	closeOrder.Topic = strings.TrimSpace(closeOrder.Topic)
	if closeOrder.ID == "" {
		resp.Write(generateFailureBody("job id不能为空"))
		return
	}
	if closeOrder.Topic == "" {
		resp.Write(generateFailureBody("job topic 不能为空"))
		return
	}

	if closeOrder.Delay <= 0 || closeOrder.Delay > (1<<31) {
		resp.Write(generateFailureBody("delay 取值范围1 - (2^31 - 1)"))
		return
	}
	closeOrder.Delay = time.Now().Unix() + closeOrder.Delay

	if closeOrder.TTR <= 0 || closeOrder.TTR > 86400 {
		resp.Write(generateFailureBody("ttr 取值范围1 - 86400"))
		return
	}

	//记录当前添加的job信息
	log.Printf("add job:%+v\n", closeOrder)

	err = delayjob.Push(closeOrder)

	if err != nil {
		log.Printf("添加job失败#%s", err.Error())
		resp.Write(generateFailureBody(err.Error()))
	} else {
		resp.Write(generateSuccessBody("添加成功", nil))
	}
}

// ResponseBody 响应Body格式
type ResponseBody struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func readBody(resp http.ResponseWriter, req *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("读取body错误#%s", err.Error())
		resp.Write(generateFailureBody("读取request body失败"))
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		log.Printf("解析json失败#%s", err.Error())
		resp.Write(generateFailureBody("解析json失败"))
		return err
	}

	log.Printf("the body is:%v\n", string(body))
	return nil
}

func generateSuccessBody(msg string, data interface{}) []byte {
	return generateResponseBody(0, msg, data)
}

func generateFailureBody(msg string) []byte {
	return generateResponseBody(1, msg, nil)
}

func generateResponseBody(code int, msg string, data interface{}) []byte {
	body := &ResponseBody{}
	body.Code = code
	body.Message = msg
	body.Data = data

	bytes, err := json.Marshal(body)
	if err != nil {
		log.Printf("生成response body,转换json失败#%s", err.Error())
		return []byte(`{"code":"1", "message": "生成响应body异常", "data":[]}`)
	}

	return bytes
}
