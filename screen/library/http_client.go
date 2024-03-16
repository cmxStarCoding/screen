package library

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func SendGETRequest(url string) ([]byte, error) {
	// 发送 GET 请求
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func SendPOSTRequest(url string, postData map[string]interface{}) ([]byte, error) {
	// 将 JSON 数据编码为字节切片
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	// 发送 POST 请求
	response, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
