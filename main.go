package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/astenir/crawler/collect"
)

func main() {
	url := "https://book.douban.com/subject/30137806/"
	var f collect.Fetcher = collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
	}
	body, err := f.Get(url)
	if err != nil {
		fmt.Printf("read content failed:%v", err)
		return
	}
	// fmt.Println(string(body))

	// 加载HTML文档
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("read content failed:%v", err)
	}

	doc.Find("p.comment-content span.short").Each(func(i int, s *goquery.Selection) {
		// 获取匹配元素的文本
		title := s.Text()
		fmt.Printf("Review %d: %s\n", i+1, title)
	})
}
