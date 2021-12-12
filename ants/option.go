package ants

type Option func(opts *Options)

type Options struct {
    SubmitNonBlock  bool      // 当协程池满了之后，如何处理，默认是阻塞处理，即会一直等待
    SubmitRetryIntervalMs int // 当协程池满了之后，如果是非阻塞，则定期重新尝试submit
}

func reloadOptions(options ...Option) *Options {
    opts := new(Options)
    for _, option := range options {
        option(opts)
    }
    return opts
}

func WithSubmitNonBlock(nonblocking bool) Option {
    return func(opts *Options) {
        opts.SubmitNonBlock = nonblocking
    }
}

func WithSubmitRetryIntervalMs(interval int) Option {
    return func(opts *Options) {
        opts.SubmitRetryIntervalMs = interval
    }
}