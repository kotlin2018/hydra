package servers

import (
	"bufio"
	"fmt"
	"io"
	"net"
	orhttp "net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/micro-plat/hydra"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/global"
	"github.com/micro-plat/hydra/hydra/servers/http"
	"github.com/micro-plat/hydra/hydra/servers/pkg/middleware"
	"github.com/micro-plat/hydra/test/assert"
	"github.com/micro-plat/hydra/test/mocks"
	"github.com/micro-plat/lib4go/errs"
)

var script0 = `

request := import("request")
app := import("app")
text := import("text")
types :=import("types")

getUpCluster := func(){
    return ""
}

upcluster := getUpCluster()`

var script1 = `

request := import("request")
app := import("app")
text := import("text")
types :=import("types")

getUpCluster := func(){
    ip := request.getClientIP()
    current:= app.getCurrentClusterName()
    if text.has_prefix(ip,"192.168."){
        return "newporxy"
    }
    return current
}

upcluster := getUpCluster()`

var script2 = `
response := import("response")

getContent := func(){
	return response.getContent1()
}

render := getContent()`

var script3 = `
getContent := func(){
	return [error]
}

render := getContent()`

//author:taosy
//time:2020-11-18
//desc:测试灰度中间件逻辑
func TestProxy(t *testing.T) {

	startUpstreamServer(":5121")
	type testCase struct {
		name            string
		isSet           bool
		script          string
		requestURL      string
		localIP         string
		Status          int
		CType           string
		Content         string
		wantStatus      int
		wantContent     string
		wantSpecial     string
		wantContentType string
	}

	tests := []*testCase{
		// {name: "proxy-没有设置节点", isSet: false, script: "", requestURL: "", localIP: "", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 200, wantContent: "success", wantContentType: "application/xml", wantSpecial: ""},
		// {name: "proxy-设置错误的节点,编译失败", isSet: true, script: script3, requestURL: "", localIP: "", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 510, wantContent: "acl.proxy脚本错误", wantContentType: "application/xml", wantSpecial: ""},
		// {name: "proxy-设置错误的节点,运行失败", isSet: true, script: script2, requestURL: "", localIP: "", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 502, wantContent: "", wantContentType: "application/xml", wantSpecial: "proxy"},
		// {name: "proxy-设置正确的节点,就是当前集群", isSet: true, script: script1, requestURL: "", localIP: "192.167.0.111", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 200, wantContent: "success", wantContentType: "application/xml", wantSpecial: ""},
		// {name: "proxy-设置正确的节点,上游集群名为空", isSet: true, script: script0, requestURL: "", localIP: "192.168.0.111", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 200, wantContent: "success", wantContentType: "application/xml", wantSpecial: ""},
		// {name: "proxy-设置正确的节点,不是api或rpc服务", isSet: true, script: script1, requestURL: "", localIP: "192.168.0.111", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 502, wantContent: "只有api,web服务器支持代理配置", wantContentType: "application/xml", wantSpecial: "proxy"},
		// {name: "proxy-设置正确的节点,上游集群无服务", isSet: true, script: script1, requestURL: "", localIP: "192.168.0.111", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 502, wantContent: "无法获取上游服务器地址", wantContentType: "application/xml", wantSpecial: "proxy"},
		{name: "proxy-设置正确的节点,上游集群存在,服务器返回异常", isSet: true, script: script1, requestURL: "/upcluster/err", localIP: "192.168.0.111", Status: 200, Content: "success", CType: "application/xml",
			wantStatus: 502, wantContent: "远程请求出错", wantContentType: "application/xml", wantSpecial: "proxy"},
		// {name: "proxy-设置正确的节点,上游集群存在,服务被拒绝", isSet: true, script: script2, requestURL: "", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 200, wantContent: "success", wantContentType: "application/xml", wantSpecial: ""},
		// {name: "proxy-设置正确的节点,上游集群存在,服务可用", isSet: true, script: script2, requestURL: "", Status: 200, Content: "success", CType: "application/xml",
		// 	wantStatus: 200, wantContent: "success", wantContentType: "application/xml", wantSpecial: ""},
	}
	for _, tt := range tests {
		global.Def.ServerTypes = []string{http.API}
		conf := mocks.NewConfBy("middleware_test", "porxy")
		confN := conf.API(":5120")
		if tt.isSet {
			confN.Proxy(tt.script)
		}

		req, _ := orhttp.NewRequest("GET", tt.localIP, nil)

		req.Header = map[string][]string{}
		//初始化测试用例参数
		ctx := &mocks.MiddleContext{
			MockUser:     &mocks.MockUser{MockClientIP: tt.localIP},
			MockRequest:  &mocks.MockRequest{MockPath: &mocks.MockPath{MockRequestPath: tt.requestURL}},
			MockResponse: &mocks.MockResponse{MockStatus: tt.Status, MockContent: tt.Content, MockHeader: map[string][]string{"Content-Type": []string{tt.CType}}},
			MockAPPConf:  conf.GetAPIConf(),
			HttpRequest:  req,
			HttpResponse: &MockResponseWriter{},
		}

		//调用中间件
		gid := global.GetGoroutineID()
		context.Del(gid)
		context.Cache(ctx)
		handler := middleware.Proxy()
		handler(ctx)

		gotStatus, gotContent := ctx.Response().GetFinalResponse()
		assert.Equalf(t, tt.wantStatus, gotStatus, tt.name)
		assert.Equalf(t, true, strings.Contains(gotContent, tt.wantContent), tt.name)
		gotHeaders := ctx.Response().GetHeaders()
		assert.Equalf(t, tt.wantContentType, gotHeaders["Content-Type"][0], tt.name)

		if tt.wantSpecial != "" {
			gotSpecial := ctx.Response().GetSpecials()
			assert.Equalf(t, tt.wantSpecial, gotSpecial, tt.name)
		}
	}
}

var oncelock sync.Once

func startUpstreamServer(port string) {
	oncelock.Do(func() {
		app := hydra.NewApp(
			hydra.WithPlatName("middleware_test"),
			hydra.WithSystemName("apiserver"),
			hydra.WithServerTypes(http.API),
			hydra.WithClusterName("newporxy"),
			hydra.WithRegistry("lm://."),
		)
		hydra.Conf.API(port)
		app.API("/upcluster/ok", upclusterOK)
		app.API("/upcluster/err", upclusterErr)
		app.API("/upcluster/err1", upclusterErr1)

		os.Args = []string{"upclusterserver", "run"}
		go app.Start()
		time.Sleep(time.Second * 5)
	})
}

func upclusterOK(ctx hydra.IContext) interface{} {
	return "success"
}

func upclusterErr(ctx hydra.IContext) interface{} {
	return errs.NewError(555, fmt.Errorf("代理返回错误"))
}

func upclusterErr1(ctx hydra.IContext) interface{} {
	return errs.NewError(511, fmt.Errorf("connect: connection refused"))
}

var _ orhttp.ResponseWriter = &MockResponseWriter{}

type MockResponseWriter struct {
	orhttp.ResponseWriter
	size   int
	status int
}

func (w *MockResponseWriter) reset(writer orhttp.ResponseWriter) {
	w.ResponseWriter = writer
	w.size = -1
	w.status = orhttp.StatusOK
}

func (w *MockResponseWriter) Header() orhttp.Header {
	return map[string][]string{}
}

func (w *MockResponseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		if w.Written() {
			// debugPrint("[WARNING] Headers were already written. Wanted to override status code %d with %d", w.status, code)
		}
		w.status = code
	}
}

func (w *MockResponseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *MockResponseWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	// n, err = w.ResponseWriter.Write(data)
	// w.size += n
	return
}

func (w *MockResponseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return
}

func (w *MockResponseWriter) Status() int {
	return w.status
}

func (w *MockResponseWriter) Size() int {
	return w.size
}

func (w *MockResponseWriter) Written() bool {
	return w.size != -1
}

// Hijack implements the http.Hijacker interface.
func (w *MockResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(orhttp.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotify interface.
func (w *MockResponseWriter) CloseNotify() <-chan bool {
	return nil
}

// Flush implements the http.Flush interface.
func (w *MockResponseWriter) Flush() {
	w.WriteHeaderNow()
	w.ResponseWriter.(orhttp.Flusher).Flush()
}

func (w *MockResponseWriter) Pusher() (pusher orhttp.Pusher) {
	if pusher, ok := w.ResponseWriter.(orhttp.Pusher); ok {
		return pusher
	}
	return nil
}
