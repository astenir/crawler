package engine

import (
	"github.com/astenir/crawler/collect"
	"go.uber.org/zap"
)

type Schedule struct {
	requestCh chan *collect.Request
	workerCh  chan *collect.Request
	out       chan collect.ParseResult
	options
}

type Config struct {
	WorkCount int
	Fetcher   collect.Fetcher
	Logger    *zap.Logger
	Seeds     []*collect.Request
}

func NewSchedule(opts ...Option) *Schedule {
	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}
	s := &Schedule{}
	s.options = options
	return s
}

// Run 是 ScheduleEngine 的主控制函数，负责初始化各个 channel 和启动必要的协程。
func (s *Schedule) Run() {
	// 创建请求通道，用于接收新的请求。
	requestCh := make(chan *collect.Request)
	// 创建工作通道，用于分发请求给工作协程。
	workerCh := make(chan *collect.Request)
	// 创建输出通道，用于接收解析后的结果。
	out := make(chan collect.ParseResult)

	// 将通道赋值给 ScheduleEngine 的对应字段，以便在其他方法中使用。
	s.requestCh = requestCh
	s.workerCh = workerCh
	s.out = out

	// 启动调度协程，负责根据策略调度请求。
	go s.Schedule()

	// 根据配置的工作协程数量启动相应数量的协程，每个协程都会从workerCh中接收任务并处理。
	for i := 0; i < s.WorkCount; i++ {
		go s.CreateWork()
	}

	// 启动处理结果的协程，负责从out通道中接收解析结果并进行后续处理。
	s.HandleResult()
}

func (s *Schedule) Schedule() {
	var reqQueue = s.Seeds
	go func() {
		for {
			var req *collect.Request
			var ch chan *collect.Request

			// 如果请求队列不为空，取出第一个请求并更新队列。
			if len(reqQueue) > 0 {
				req = reqQueue[0]
				reqQueue = reqQueue[1:]
				// 将请求发送到工作者通道，准备分配给工作者进行处理。
				ch = s.workerCh
			}
			// 使用select语句来处理请求通道中的新请求或分发当前请求给工作者。
			select {
			// 当请求通道中有新请求到达时，将其加入请求队列。
			case r := <-s.requestCh:
				reqQueue = append(reqQueue, r)
			// 将当前请求分发给工作者进行处理。
			case ch <- req:
			}
		}
	}()
}

func (s *Schedule) CreateWork() {
	for {
		r := <-s.workerCh
		body, err := s.Fetcher.Get(r)
		if len(body) < 6000 {
			s.Logger.Error("can't fetch ",
				zap.Int("length", len(body)),
				zap.String("url", r.Url),
			)
		}
		if err != nil {
			s.Logger.Error("can't fetch ",
				zap.Error(err),
				zap.String("url", r.Url),
			)
			continue
		}
		result := r.ParseFunc(body, r)
		s.out <- result
	}
}

func (s *Schedule) HandleResult() {
	for result := range s.out {
		for _, req := range result.Requests {
			s.requestCh <- req
		}
		for _, item := range result.Items {
			// todo: store
			s.Logger.Sugar().Info("get result", item)
		}
	}
}
