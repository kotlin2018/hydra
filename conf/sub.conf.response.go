package conf

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/micro-plat/lib4go/concurrent/cmap"
	"github.com/micro-plat/lib4go/security/md5"
)

type Template struct {
	Params   map[string]interface{} `json:"params,omitempty"`
	Template string                 `json:"template,omitempty" valid:"required"`
	Services []string               `json:"services,omitempty" valid:"required"`
}

//Response 请求响应配置
type Response struct {
	Params    map[string]interface{} `json:"params,omitempty"`
	Templates []*Template            `json:"templates,omitempty" valid:"required"`
}

//NewResponse 构建响应配置
func NewResponse(template string, service ...string) *Response {
	r := &Response{Templates: make([]*Template, 0, 2), Params: make(map[string]interface{})}
	r.Append(template, service...)
	return r
}

//Append 追加模板配置
func (r *Response) Append(t string, service ...string) *Response {
	if _, err := template.New("").Parse(t); err != nil {
		panic(fmt.Errorf("response响应模板格式错误:%v", err))
	}
	services := service
	if len(service) == 0 {
		services = []string{"*"}
	}
	r.Templates = append(r.Templates, &Template{Template: t, Services: services})
	return r
}

//SetParam 追加模板配置
func (r *Response) SetParam(k string, v interface{}) *Response {
	r.Params[k] = v
	return r
}

//Translate 翻译模板
func (r *Template) Translate(service string, input interface{}) (string, error) {
	tmpl, err := getTemplate(r.Template)
	if err != nil {
		return "", err
	}
	buff := bytes.NewBufferString("")
	err = tmpl.Execute(buff, input)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

//GetTemplate 获取指定请求对应的模板
func (r *Response) GetTemplate(s string) (bool, *Template) {
	var last *Template
	for _, t := range r.Templates {
		for _, service := range t.Services {
			if service == s {
				last = t
				goto LOOP
			}
			if service == "*" {
				last = t
			}
		}
	}
LOOP:
	if last == nil {
		return false, nil
	}
	if last.Params == nil {
		last.Params = make(map[string]interface{})
	}
	for k, v := range r.Params {
		if _, ok := last.Params[k]; !ok {
			last.Params[k] = v
		}
	}
	return true, last
}

var tmpCache cmap.ConcurrentMap

func init() {
	tmpCache = cmap.New(2)
}

//getTemplate 获取模板信息
func getTemplate(ts string) (*template.Template, error) {
	key := md5.Encrypt(ts)
	_, tmp, err := tmpCache.SetIfAbsentCb(key, func(input ...interface{}) (c interface{}, err error) {
		t := input[0].(string)
		tmpl, err := template.New(key).Parse(t)
		return tmpl, err

	}, ts)
	if err != nil {
		return nil, err
	}
	return tmp.(*template.Template), nil
}
