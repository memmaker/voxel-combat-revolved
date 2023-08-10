package util

import (
	"encoding/json"
	"fmt"
	"time"
)

func FromJson(data string, msg any) bool {
	err := json.Unmarshal([]byte(data), msg)
	if err != nil {
		println(fmt.Sprintf("FATAL JSON UNMARSHAL ERROR: %s", err.Error()))
		return false
	}
	return true
}

func WaitForTrue(success *bool) {
	for !*success {
		time.Sleep(250 * time.Millisecond)
	}
}

func MustSend(err error) {
	if err != nil {
		panic(fmt.Sprintf("FATAL CONNECTION ERROR: %s", err.Error()))
	}
}
