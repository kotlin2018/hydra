package ctx

import (
	r "context"
	"sync"
	"time"

	"github.com/micro-plat/hydra/conf"
	"github.com/micro-plat/hydra/conf/server"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/lib4go/logger"
)

var _ context.IContext = &Ctx{}
var contextPool *sync.Pool

func init() {
	contextPool = &sync.Pool{
		New: func() interface{} {
			return &Ctx{}
		},
	}
}

//Ctx context
type Ctx struct {
	context    context.IInnerContext
	ctx        r.Context
	meta       conf.IMeta
	log        logger.ILogger
	request    *request
	response   *response
	user       *user
	serverConf server.IServerConf
	cancelFunc func()
	funs       *tmplFuncs
	tid        uint64
}

//NewCtx 构建基于gin.Context的上下文
func NewCtx(c context.IInnerContext, tp string) *Ctx {
	ctx := contextPool.Get().(*Ctx)
	ctx.meta = conf.NewMeta()
	ctx.funs = newTmplFunc(ctx)
	ctx.context = c
	var err error
	ctx.serverConf, err = server.Cache.GetServerConf(tp)
	if err != nil {
		panic(err)
	}
	ctx.user = newUser(c, ctx.meta)
	ctx.request = newRequest(c, ctx.serverConf, ctx.meta)
	ctx.log = logger.GetSession(ctx.serverConf.GetMainConf().GetServerName(), ctx.User().GetRequestID())
	ctx.response = newResponse(c, ctx.serverConf, ctx.log, ctx.meta)
	ctx.tid = context.Cache(ctx) //保存到缓存中
	timeout := time.Duration(ctx.serverConf.GetMainConf().GetMainConf().GetInt("", 30))
	ctx.ctx, ctx.cancelFunc = r.WithTimeout(r.WithValue(r.Background(), "X-Request-Id", ctx.user.GetRequestID()), time.Second*timeout)
	return ctx
}

//Meta 获取元数据配置
func (r *Ctx) Meta() conf.IMeta {
	return r.meta
}

//Request 获取请求对象
func (c *Ctx) Request() context.IRequest {
	return c.request
}

//TmplFuncs 提供用于模板转换的函数表达式
func (c *Ctx) TmplFuncs() context.TFuncs {
	return c.funs.Instance()
}

//Response 获取响应对象
func (c *Ctx) Response() context.IResponse {
	return c.response
}

//Context 处理程序退出，超时等
func (c *Ctx) Context() r.Context {
	return c.ctx
}

//User 获取用户相关信息
func (c *Ctx) User() context.IUser {
	return c.user
}

//Log 获取日志组件
func (c *Ctx) Log() logger.ILogger {
	return c.log
}

//ServerConf 获取服务器配置
func (c *Ctx) ServerConf() server.IServerConf {
	return c.serverConf
}

//Close 关闭并释放所有资源
func (c *Ctx) Close() {
	context.Del(c.tid) //从当前请求上下文中删除
	c.context = nil
	c.serverConf = nil
	c.user = nil
	c.response = nil
	c.request = nil
	c.cancelFunc()
	c.ctx = nil

	contextPool.Put(c)
}
