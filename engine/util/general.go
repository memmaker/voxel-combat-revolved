package util

import (
	"encoding/json"
	"time"
)

func FromJson(data string, msg any) bool {
	err := json.Unmarshal([]byte(data), msg)
	if err != nil {
		println(err.Error())
		return false
	}
	return true
}

func WaitForTrue(success *bool) {
	for !*success {
		time.Sleep(1 * time.Second)
	}
}

func MustSend(err error) {
	if err != nil {
		println(err.Error())
		panic(err)
	}
}
