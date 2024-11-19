package main

import (
	"fmt"
	"time"

	"github.com/astenir/crawler/collect"
	"github.com/astenir/crawler/log"
	"github.com/astenir/crawler/parse/doubangroup"
	"github.com/astenir/crawler/proxy"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	// log
	// plugin, c := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	// defer c.Close()
	plugin := log.NewStdoutPlugin(zapcore.InfoLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// proxy
	proxyURLs := []string{}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
	}

	// cookie

	// url
	var worklist []*collect.Request
	for i := 0; i <= 0; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d&type=new", i)
		worklist = append(worklist, &collect.Request{
			Url:       str,
			ParseFunc: doubangroup.ParseURL,
		})
	}

	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Proxy:   p,
	}

	for len(worklist) > 0 {
		items := worklist
		worklist = nil
		for _, item := range items {
			body, err := f.Get(item)
			time.Sleep(1 * time.Second)
			if err != nil {
				logger.Error("read content failed",
					zap.Error(err),
				)
				continue
			}
			res := item.ParseFunc(body, item)
			for _, item := range res.Items {
				logger.Info("result",
					zap.String("get url:", item.(string)))
			}
			worklist = append(worklist, res.Requests...)
		}
	}

}
