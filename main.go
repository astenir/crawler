package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://www.baidu.com/"
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("fetch url failed, err:%v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("bad status: %v\n", resp.StatusCode)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("read body failed, err:%v\n", err)
		return
	}
	fmt.Println("body:", string(body))
}
