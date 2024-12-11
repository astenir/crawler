package main

import (
	"context"
	"net/http"
	"time"

	pb "github.com/astenir/crawler/proto/greeter"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	_ "github.com/go-sql-driver/mysql"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go-micro.dev/v4"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/astenir/crawler/collect"
	"github.com/astenir/crawler/engine"
	"github.com/astenir/crawler/limiter"
	"github.com/astenir/crawler/log"
	"github.com/astenir/crawler/proxy"
	"github.com/astenir/crawler/storage"
	"github.com/astenir/crawler/storage/sqlstorage"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")

	// set zap global logger
	zap.ReplaceGlobals(logger)

	// proxy

	var err error

	var p proxy.Func

	proxyURLs := []string{"http://47.122.65.254:8080", "http://183.247.152.98:53281", "http://8.130.34.44:8800"}

	if p, err = proxy.RoundRobinProxySwitcher(proxyURLs...); err != nil {
		logger.Error("RoundRobinProxySwitcher failed")

		return
	}

	// fetcher
	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 30000 * time.Millisecond,
		Logger:  logger,
		Proxy:   p,
	}

	// storage
	var storage storage.Storage

	if storage, err = sqlstorage.New(
		sqlstorage.WithSQLURL("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	); err != nil {
		logger.Error("create sqlstorage failed")

		return
	}

	// speed limiter
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)
	minuteLimit := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20)
	multiLimiter := limiter.Multi(secondLimit, minuteLimit)

	// init tasks
	var seeds = make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
		Limit:   multiLimiter,
	})

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	// worker start
	go s.Run()

	// start http proxy to GRPC
	go RunHTTPServer()

	// start grpc server
	RunGRPCServer(logger)
}

func RunGRPCServer(logger *zap.Logger) {
	reg := etcdReg.NewRegistry(registry.Addrs(":2379"))
	service := micro.NewService(
		micro.Server(gs.NewServer(
			server.Id("1"),
		)),
		micro.Address(":9090"),
		micro.Registry(reg),
		micro.RegisterTTL(60*time.Second),
		micro.RegisterInterval(15*time.Second),
		micro.WrapHandler(logWrapper(logger)),
		micro.Name("go.micro.server.worker"),
	)
	// 设置micro 客户端默认超时时间为10秒钟
	if err := service.Client().Init(client.RequestTimeout(10 * time.Second)); err != nil {
		logger.Sugar().Error("micro client init error. ", zap.String("error:", err.Error()))

		return
	}

	service.Init()

	if err := pb.RegisterGreeterHandler(service.Server(), new(Greeter)); err != nil {
		logger.Fatal("register handler failed")
	}

	if err := service.Run(); err != nil {
		logger.Fatal("grpc server stop")
	}
}

type Greeter struct{}

func (g *Greeter) Hello(_ context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.GetName()

	return nil
}
func RunHTTPServer() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:9090", opts); err != nil {
		zap.L().Fatal("Register backend grpc server endpoint failed")
	}

	if err := http.ListenAndServe(":8080", mux); err != nil {
		zap.L().Fatal("http listenAndServe failed")
	}
}

func logWrapper(log *zap.Logger) server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			log.Info("receive request",
				zap.String("method", req.Method()),
				zap.String("Service", req.Service()),
				zap.Reflect("request param:", req.Body()),
			)

			err := fn(ctx, req, rsp)

			return err
		}
	}
}
