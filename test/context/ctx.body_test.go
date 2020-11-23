package context

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/micro-plat/hydra/conf"
	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/context/ctx"
	"github.com/micro-plat/hydra/test/assert"
	"github.com/micro-plat/hydra/test/mocks"
	"github.com/micro-plat/lib4go/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type testBody struct {
	name    string
	wantS   string
	wantErr bool
	err     error
}

func Test_body_GetBody_MIMEXML(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "application/xml", contentType: "application/xml",
			body: `<?xml version="1.0"?><methodCall><methodName>examples.getStateName</methodName><params><param><i4>41</i4></param></params></methodCall>`,
			want: `<methodCall><methodName>examples.getStateName</methodName><params><param><i4>41</i4></param></params></methodCall>`},
		{name: "text/xml", contentType: "text/xml", body: "<xml><sub>1</sub></xml>", want: "<xml><sub>1</sub></xml>"},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbodymap", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, false, err != nil, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, tt.contentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(body), tt.name)
	}
}

func Test_body_GetBody_MIMEJSON(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "application/json", contentType: "application/json",
			body: `{"name":"BeJson","page":88,"isNonProfit":true,"address":{"street":"科技园路.","city":"江苏苏州","country":"中国"}}`,
			want: `{"name":"BeJson","page":88,"isNonProfit":true,"address":{"street":"科技园路.","city":"江苏苏州","country":"中国"}}`},
		{name: "text/json", contentType: "text/json",
			body: `{"name":"BeJson","page":88,"isNonProfit":true,"address":{"street":"科技园路.","city":"江苏苏州","country":"中国"}}`,
			want: `{"name":"BeJson","page":88,"isNonProfit":true,"address":{"street":"科技园路.","city":"江苏苏州","country":"中国"}}`},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbodymap", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, false, err != nil, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, tt.contentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		want := map[string]interface{}{}
		json.Unmarshal([]byte(tt.want), &want)

		got := map[string]interface{}{}
		json.Unmarshal(body, &got)
		assert.Equal(t, want, got, tt.name)
	}
}

func Test_body_GetBody_MIMEYAML(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "text/yaml", contentType: "text/yaml; charset=utf-8", body: `animal: pets`, want: "animal: pets\n"},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbodymap", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, nil, err, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, tt.contentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(body), tt.name)
	}
}
func Test_body_GetBody_MIMEPlain(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "text/plain", contentType: "text/plain; charset=utf-8", body: `body`, want: "map[__body_:body]"},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbodymap", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, nil, err, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, tt.contentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(body), tt.name)
	}
}
func Test_body_GetBody_MIMEPOSTForm(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		{name: "application/x-www-form-urlencoded", contentType: "application/x-www-form-urlencoded; charset=utf-8",
			body: `a=1&b=2&c=3`, want: "a=1&b=2&c=3"},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/form", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, nil, err, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, "text/plain", resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(body), tt.name)
	}
}

func Test_body_GetBody_MIMEMultipartPOSTForm(t *testing.T) {
	tests := []struct {
		name string
		url  string
		path string
		want string
	}{
		{name: "multipart/form-data", url: "http://localhost:9091/upload", path: "upload.test.txt",
			want: `{"fileName":"upload.test.txt","size":25,"body":"ADASDASDASFHNOJM~!@#$%^&*"}`},
	}
	startServer()
	for _, tt := range tests {
		file, _ := os.Open(tt.path)
		defer file.Close()
		body := &bytes.Buffer{}
		// 文件写入 body
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("upload", filepath.Base(tt.path))
		assert.Equal(t, nil, err, "文件写入 body1")
		_, err = io.Copy(part, file)
		assert.Equal(t, nil, err, "文件写入 body2")
		err = writer.Close()
		assert.Equal(t, nil, err, "文件写入 body3")
		req, err := http.NewRequest(http.MethodPost, tt.url, body)
		assert.Equal(t, nil, err, "创建请求1")
		req.Header.Add("Content-Type", writer.FormDataContentType())
		client := &http.Client{}
		resp, err := client.Do(req)
		assert.Equal(t, nil, err, tt.name)
		defer resp.Body.Close()
		assert.Equal(t, "application/json", resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, "200 OK", resp.Status, tt.name)
		assert.Equal(t, 200, resp.StatusCode, tt.name)
		rbody, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		w := map[string]interface{}{}
		json.Unmarshal([]byte(tt.want), &w)
		r := map[string]interface{}{}
		json.Unmarshal(rbody, &r)
		assert.Equal(t, w, r, tt.name)
	}
}
func Test_body_GetBody(t *testing.T) {
	//测试读取body正确
	tests := []testBody{
		{name: "1-首次读取body且读取和解码无错误", wantS: "  body", wantErr: false},
		{name: "1-再次读取body且返回body", wantS: "  body", wantErr: false},
	}
	testGetBody(t, "%20+body", tests)

	//测试读取错误
	testReadErr := []testBody{
		{name: "2-首次读取body且读取错误", wantS: "", wantErr: true, err: fmt.Errorf("获取body发生错误:读取出错")},
		{name: "2-再次读取body且返回的读取错误", wantS: "", wantErr: true, err: fmt.Errorf("读取出错")},
	}
	testGetBody(t, "TEST_BODY_READ_ERR", testReadErr)

	//测试解码错误
	testUnescapeErr := []testBody{
		{name: "3-首次读取body,读取正确,解码错误", wantS: "", wantErr: true, err: fmt.Errorf(`url.unescape出错:invalid URL escape "%%-+"`)},
		{name: "3-再次读取body且返回的解码错误", wantS: "", wantErr: true, err: fmt.Errorf(`invalid URL escape "%%-+"`)},
	}
	testGetBody(t, "%-+body", testUnescapeErr)

}

func testGetBody(t *testing.T, body string, tests []testBody) {
	confObj := mocks.NewConfBy("context_body_test", "bodyctx") //构建对象
	confObj.API(":8080")                                       //初始化参数
	serverConf := confObj.GetAPIConf()                         //获取配置
	rpath := ctx.NewRpath(&mocks.TestContxt{}, serverConf, conf.NewMeta())
	w := ctx.NewBody(&mocks.TestContxt{Body: body}, rpath.GetEncoding())

	for _, tt := range tests {
		gotS, err := w.GetBody()
		if (err != nil) == tt.wantErr && tt.err != nil {
			assert.Equal(t, tt.err.Error(), err.Error(), tt.name)
		}
		assert.Equal(t, tt.wantErr, err != nil, tt.name)
		assert.Equal(t, tt.wantS, gotS, tt.name)
	}
}

func Test_body_GetBodyMap_WithPanic(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.IInnerContext
		encoding []string
		want     map[string]interface{}
		err      string
	}{
		{name: "解析yaml数据错误", ctx: &mocks.TestContxt{
			Body:       `body`,
			HttpHeader: http.Header{"Content-Type": []string{context.YAMLF}},
		},
			err: "将body转换为map失败:yaml: unmarshal errors:\n  line 1: cannot unmarshal !!str `body` into map[string]interface {}"},
		{name: "解析xml数据错误", ctx: &mocks.TestContxt{
			Body:       `body`,
			HttpHeader: http.Header{"Content-Type": []string{context.XMLF}},
		}, err: "将body转换为map失败:EOF"},
		{name: "解析json数据错误", ctx: &mocks.TestContxt{
			Body:       `body`,
			HttpHeader: http.Header{"Content-Type": []string{context.JSONF}},
		}, err: "将body转换为map失败:invalid character 'b' looking for beginning of value"},
	}

	confObj := mocks.NewConfBy("context_body_test1", "bodyctx1") //构建对象
	confObj.API(":8080")                                         //初始化参数
	serverConf := confObj.GetAPIConf()                           //获取配置
	rpath := ctx.NewRpath(&mocks.TestContxt{}, serverConf, conf.NewMeta())

	for _, tt := range tests {
		w := ctx.NewBody(tt.ctx, rpath.GetEncoding())
		assert.PanicError(t, tt.err, func() {
			w.GetBodyMap()
		}, tt.name)
	}
}

func Test_body_GetBodyMap(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.IInnerContext
		encoding []string
		want     map[string]interface{}
		wantErr  bool
	}{
		{name: "getBody报错", ctx: &mocks.TestContxt{
			Body:       `"%-+body"`,
			HttpHeader: http.Header{"Content-Type": []string{context.JSONF}},
		}, wantErr: true},
		{name: "content-type为application/xml", ctx: &mocks.TestContxt{
			Body:       `<xml><sub z="3"><sg a="1" b="2">12</sg></sub></xml>`,
			HttpHeader: http.Header{"Content-Type": []string{context.XMLF}},
		}, encoding: []string{"gbk"}, want: map[string]interface{}{
			"xml": map[string]interface{}{
				"sub": map[string]interface{}{
					"z": "3",
					"sg": map[string]interface{}{
						"a":     "1",
						"b":     "2",
						"#text": "12",
					}}}}},
		{name: "content-type为text/xml", ctx: &mocks.TestContxt{
			Body:       `<xml><key1>1&amp;$</key1><key2>value2</key2></xml>`,
			HttpHeader: http.Header{"Content-Type": []string{"text/xml"}},
		}, encoding: []string{}, want: map[string]interface{}{
			"xml": map[string]interface{}{
				"key1": "1&$", "key2": "value2",
			}}},
		{name: "content-type为application/json", ctx: &mocks.TestContxt{
			Body:       `{"key1":"value1","key2":"value2"}`,
			HttpHeader: http.Header{"Content-Type": []string{context.JSONF}},
		}, want: map[string]interface{}{"key1": "value1", "key2": "value2"}},
		{name: "content-type为text/json", ctx: &mocks.TestContxt{
			Body:       `{"key1":"value1","key2":"value2"}`,
			HttpHeader: http.Header{"Content-Type": []string{"text/json"}},
		}, want: map[string]interface{}{"key1": "value1", "key2": "value2"}},
		{name: "content-type为text/yaml", ctx: &mocks.TestContxt{
			Body:       `key1: value1`,
			HttpHeader: http.Header{"Content-Type": []string{context.YAMLF}},
		}, want: map[string]interface{}{"key1": "value1"}},
	}

	confObj := mocks.NewConfBy("context_body_test2", "bodyctx2") //构建对象
	confObj.API(":8080")                                         //初始化参数
	serverConf := confObj.GetAPIConf()                           //获取配置
	rpath := ctx.NewRpath(&mocks.TestContxt{}, serverConf, conf.NewMeta())

	for _, tt := range tests {
		w := ctx.NewBody(tt.ctx, rpath.GetEncoding())
		got, err := w.GetBodyMap()
		assert.Equal(t, tt.wantErr, err != nil, tt.name)
		assert.Equal(t, tt.want, got, tt.name)
	}
}

func getTestUTF8Json(s map[string]string) string {
	for k, v := range s {
		s[k] = url.QueryEscape(v)
	}
	r, _ := json.Marshal(s)
	return string(r)
}

func getTestGBKJson(s map[string]string) string {
	for k, v := range s {
		s[k] = url.QueryEscape(Utf8ToGbk(v))
	}

	r, _ := json.Marshal(s)
	return string(r)
}

func Utf8ToGbk(s string) string {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewEncoder())
	d, _ := ioutil.ReadAll(reader)
	return string(d)
}

func GbkToUtf8(s string) string {
	reader := transform.NewReader(bytes.NewReader([]byte(s)), simplifiedchinese.GBK.NewDecoder())
	d, _ := ioutil.ReadAll(reader)
	return string(d)
}

func Test_body_GetBody_Encoding(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		encoding        string
		body            string
		want            string
		wantContentType string
		wantStatus      string
		wantStatusCode  int
	}{
		{name: "头部编码utf-8,请求数据为utf-8", contentType: "application/json; charset=utf-8", encoding: "utf-8", wantContentType: "application/json; charset=utf-8",
			body: getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`, wantStatus: "200 OK", wantStatusCode: 200},
		{name: "头部编码utf-8,请求数据为gbk", contentType: "application/json; charset=gbk", encoding: "gbk", wantContentType: "application/json; charset=gbk",
			body: getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`, wantStatus: "200 OK", wantStatusCode: 200},
		{name: "头部编码gbk,请求数据为utf-8", contentType: "application/json; charset=gbk", encoding: "utf-8", wantContentType: "text/plain; charset=gbk",
			body: getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}), wantStatus: "510 Not Extended", wantStatusCode: 510,
			want: `Server Error`},
		{name: "头部编码gbk,请求数据为gbk", contentType: "application/json; charset=gbk", encoding: "gbk", wantContentType: "application/json; charset=gbk",
			body: getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`, wantStatus: "200 OK", wantStatusCode: 200},
		{name: "未设置charset,请求数据为gbk", contentType: "application/json", encoding: "utf-8", wantContentType: "application/json; charset=utf-8",
			body: getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`), wantStatus: "200 OK", wantStatusCode: 200},
		{name: "未设置charset,请求数据为utf-8", contentType: "application/json; charset=utf-8", encoding: "utf-8", wantContentType: "application/json; charset=utf-8",
			body: getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`, wantStatus: "200 OK", wantStatusCode: 200},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbody/encoding", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, false, err != nil, tt.name)

		assert.Equal(t, tt.wantContentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.wantStatus, resp.Status, tt.name)
		assert.Equal(t, tt.wantStatusCode, resp.StatusCode, tt.name)

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		got, _ := encoding.Decode(string(body), tt.encoding)
		assert.Equal(t, tt.want, string(got), tt.name)
	}
}

func Test_body_GetBody_Encoding_UTF8(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		encoding        string
		body            string
		want            string
		wantStatus      string
		wantContentType string
		wantStatusCode  int
	}{
		{name: "请求编码为GBK", contentType: "application/json;charset=gbk", wantContentType: "application/json; charset=utf-8",
			body:       getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`)},
		{name: "请求数据为GBK,未设置头部编码", contentType: "application/json", wantContentType: "application/json; charset=utf-8",
			body:       getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`)},
		{name: "请求数据为utf-8,头为gbk", contentType: "application/json;charset=gbk", wantContentType: "application/json; charset=utf-8",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`},
		{name: "请求编码为utf-8", contentType: "application/json;charset=utf-8", wantContentType: "application/json; charset=utf-8",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`},
		{name: "请求数据为utf-8,未设置头部编码", contentType: "application/json", wantContentType: "application/json; charset=utf-8",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: `{"address":"科技园路~!#$%^&*()_+{}:<?"}`},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbody/encoding/utf8", tt.contentType, strings.NewReader(tt.body))
		//fmt.Printf("resp:%+v \n", resp)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.wantContentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, tt.wantStatusCode, resp.StatusCode, tt.name)
		assert.Equal(t, tt.wantStatus, resp.Status, tt.name)
		defer resp.Body.Close()
		got, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(got), tt.name)
	}
}

func Test_body_GetBody_Encoding_GBK(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		body            string
		want            string
		wantStatus      string
		wantContentType string
		wantStatusCode  int
	}{
		{name: "请求编码为utf-8,头为gbk", contentType: "application/json;charset=gbk", wantContentType: "text/plain; charset=gbk",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "510 Not Extended", wantStatusCode: 510, want: `Server Error`},
		{name: "请求编码为utf-8", contentType: "application/json;charset=utf-8", wantContentType: "text/plain; charset=gbk",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "510 Not Extended", wantStatusCode: 510, want: "Server Error"},
		{name: "请求数据为utf-8,未设置头部编码", contentType: "application/json", wantContentType: "text/plain; charset=gbk",
			body:       getTestUTF8Json(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "510 Not Extended", wantStatusCode: 510, want: "Server Error"},
		{name: "请求数据为gbk,头为utf-8", contentType: "application/json;charset=utf-8", wantContentType: "application/json; charset=gbk",
			body:       getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`)},
		{name: "请求编码为gbk", contentType: "application/json;charset=gbk", wantContentType: "application/json; charset=gbk",
			body:       getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`)},
		{name: "请求数据为gbk,未设置头部编码", contentType: "application/json", wantContentType: "application/json; charset=gbk",
			body:       getTestGBKJson(map[string]string{"address": "科技园路~!#$%^&*()_+{}:<?"}),
			wantStatus: "200 OK", wantStatusCode: 200, want: Utf8ToGbk(`{"address":"科技园路~!#$%^&*()_+{}:<?"}`)},
	}
	startServer()
	for _, tt := range tests {
		resp, err := http.Post("http://localhost:9091/getbody/encoding/gbk", tt.contentType, strings.NewReader(tt.body))
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.wantContentType, resp.Header["Content-Type"][0], tt.name)
		assert.Equal(t, tt.wantStatusCode, resp.StatusCode, tt.name)
		assert.Equal(t, tt.wantStatus, resp.Status, tt.name)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		assert.Equal(t, false, err != nil, tt.name)
		assert.Equal(t, tt.want, string(body), tt.name)
	}
}

func Test_request_api(t *testing.T) {

	startServer()

	resp, err := http.Post("http://localhost:9091/rpc", "application/json", strings.NewReader(`{"a":1}`))
	assert.Equal(t, false, err != nil, "xxxx")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body), err)
}
