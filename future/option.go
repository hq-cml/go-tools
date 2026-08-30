package future

const (
	LogId   = "log_id"   // 日志ID、traceId
	LaneTag = "lane_tag" // 流量泳道标记
)

// Logger 日志接口，用于记录 Future 执行耗时等信息。
// 标准库 *log.Logger 即满足此接口。
type Logger interface {
	Printf(format string, args ...interface{})
}

// Option 用于配置 Future 或 FuturePool 的行为
// Option 是一个func类型，该类func用于调整futureCFG
type Option func(*futureCFG)

type futureCFG struct {
	logger  Logger
	gtxKeys map[interface{}]struct{} // 扩展：需要传递给子 goroutine 的额外 gtx keys（LogId/LaneTag已自带）
}

type noopLogger struct{}

func (noopLogger) Printf(string, ...interface{}) {}

// WithLogger 设置日志记录器
// 不设置时默认无日志输出。
func WithLogger(l Logger) Option {
	return func(c *futureCFG) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithGtxKeys 指定额外需要从父 goroutine 传递到子 goroutine 的 gtx keys。
// LogId 和 LaneTag 默认已传递，无需重复指定。
func WithGtxKeys(keys ...interface{}) Option {
	return func(c *futureCFG) {
		for _, k := range keys {
			c.gtxKeys[k] = struct{}{}
		}
	}
}

// newDefaultConfig 返回默认配置
// 默认noopLogger + 默认的 gtx keys（LogId/LaneTag）
func newDefaultConfig() *futureCFG {
	return &futureCFG{
		logger:  noopLogger{},
		gtxKeys: map[interface{}]struct{}{LogId: {}, LaneTag: {}},
	}
}

// newConfig 在默认配置基础上应用用户选项
func newConfig(opts []Option) *futureCFG {
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
