package main

import (
	"time"

	"github.com/astenir/crawler/collect"
	"github.com/astenir/crawler/collector"
	"github.com/astenir/crawler/collector/sqlstorage"
	"github.com/astenir/crawler/engine"
	"github.com/astenir/crawler/log"
	"github.com/astenir/crawler/proxy"
	"go.uber.org/zap/zapcore"
)

func main() {

	// log
	// plugin, c := log.NewFilePlugin("./log.txt", zapcore.InfoLevel)
	// defer c.Close()
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// proxy
	proxyURLs := []string{}
	p, err := proxy.RoundRobinProxySwitcher(proxyURLs...)
	if err != nil {
		logger.Error("RoundRobinProxySwitcher failed")
	}

	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 30000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	var storage collector.Storage
	storage, err = sqlstorage.New(
		sqlstorage.WithSqlUrl("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	)
	if err != nil {
		logger.Error("create sqlstorage failed")
		return
	}

	// url
	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
	})

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)
	s.Run()

}
