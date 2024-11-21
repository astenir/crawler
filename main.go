package main

import (
	"fmt"
	"time"

	"github.com/astenir/crawler/collect"
	"github.com/astenir/crawler/engine"
	"github.com/astenir/crawler/log"
	"github.com/astenir/crawler/parse/doubangroup"
	"github.com/astenir/crawler/proxy"
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

	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	// url
	var seeds = make([]*collect.Task, 0, 1000)
	for i := 0; i <= 0; i += 25 {
		str := fmt.Sprintf("https://www.douban.com/group/szsh/discussion?start=%d&type=new", i)
		seeds = append(seeds, &collect.Task{
			Url:      str,
			WaitTime: 1 * time.Second,
			Fetcher:  f,
			MaxDepth: 5,
			RootReq: &collect.Request{
				Method:    "GET",
				ParseFunc: doubangroup.ParseURL,
			},
		})
	}

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()

}
