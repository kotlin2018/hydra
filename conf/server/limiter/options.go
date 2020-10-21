package limiter

type Option func(*Limiter)

func WithEnable() Option {
	return func(l *Limiter) {
		l.Disable = false
	}
}

func WithDisable() Option {
	return func(l *Limiter) {
		l.Disable = true
	}
}

func WithRule(rules ...*Rule) Option {
	return func(l *Limiter) {
		l.Rules = append(l.Rules, rules...)
	}
}

//RuleOption 配置选项
type RuleOption func(*Rule)

//WithAction 设置请求类型
func WithAction(action ...string) RuleOption {
	return func(a *Rule) {
		a.Action = action
	}
}

//WithMaxWait 设置请求允许等待的时长
func WithMaxWait(second int) RuleOption {
	return func(a *Rule) {
		a.MaxWait = second
	}
}

//WithFallback 启用服务降级处理
func WithFallback() RuleOption {
	return func(a *Rule) {
		a.Fallback = true
	}
}

//WithReponse 设置响应内容
func WithReponse(status int, content string) RuleOption {
	return func(a *Rule) {
		a.Resp = &Resp{Status: status, Content: content}
	}
}
