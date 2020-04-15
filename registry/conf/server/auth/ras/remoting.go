package ras

import (
	"encoding/json"
)

//RASAuth 远程认证服务
type RASAuth struct {
	//远程验证服务名
	Service string `json:"service,omitempty" valid:"required"`
	*remotingOption
}

//NewRASAuth 创建远程服务验证参数
func NewRASAuth(service string, opts ...RemotingOption) *RASAuth {
	r := &RASAuth{
		Service: service,
		remotingOption: &remotingOption{
			Requests: []string{"*"},
			Connect:  &Connect{},
			Params:   make(map[string]interface{}),
			Required: make([]string, 0, 1),
			Alias:    make(map[string]string),
			Decrypt:  make([]string, 0, 1),
		},
	}
	for _, opt := range opts {
		opt(r.remotingOption)
	}
	return r
}

//String 获取签名串
func (a *RASAuth) String() (string, error) {
	buff, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

//AuthString 获取签名串
func (a *RASAuth) AuthString() (string, error) {
	b := *a
	b.Service = ""
	b.Requests = nil
	c := &b
	return c.String()
}

//RASAuths 远程服务验证组
type RASAuths []*RASAuth

//Contains 检查指定的路径是否允许签名
func (a RASAuths) Contains(p string) (bool, *RASAuth) {
	var last *RASAuth
	for _, auth := range a {
		for _, req := range auth.Requests {
			if req == p {
				return true, auth
			}
			if req == "*" {
				last = auth
			}
		}
	}
	return last != nil, last
}
