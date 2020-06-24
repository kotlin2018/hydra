package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/context/ctx"
	"github.com/micro-plat/hydra/global"
	"github.com/micro-plat/hydra/hydra/servers/pkg/dispatcher"
)

type imiddle interface {
	Next()
}

//IMiddleContext 中间件转换器，在context.IContext中扩展next函数
type IMiddleContext interface {
	imiddle
	context.IContext
	Trace(...interface{})
}

//MiddleContext 中间件转换器，在context.IContext中扩展next函数

type MiddleContext struct {
	context.IContext
	imiddle
}

//Trace 输出调试日志
func (m *MiddleContext) Trace(s ...interface{}) {
	if global.IsDebug {
		m.IContext.Log().Debug(s...)
	}
}

//newMiddleContext 构建中间件处理handler
func newMiddleContext(c context.IContext, n imiddle) IMiddleContext {
	return &MiddleContext{IContext: c, imiddle: n}
}

//Handler 通用的中间件处理服务
type Handler func(IMiddleContext)

//GinFunc 返回GIN对应的处理函数
func (h Handler) GinFunc(tps ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, ok := c.Get("__middle_context__")
		if !ok {
			nctx := ctx.NewCtx(&ginCtx{Context: c}, tps[0])
			nctx.Meta().Set("__context_", c)
			v = newMiddleContext(nctx, c)
			c.Set("__middle_context__", v)
		}
		h(v.(IMiddleContext))
	}
}

//DispFunc 返回disp对应的处理函数
func (h Handler) DispFunc(tps ...string) dispatcher.HandlerFunc {
	return func(c *dispatcher.Context) {
		v, ok := c.Get("__middle_context__")
		if !ok {
			nctx := ctx.NewCtx(&dispCtx{Context: c}, tps[0])
			nctx.Meta().Set("__context_", c)
			v = newMiddleContext(nctx, c)
			c.Set("__middle_context__", v)
		}
		h(v.(IMiddleContext))
	}
}
