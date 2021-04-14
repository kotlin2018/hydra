package nfs

import (
	"fmt"
	"hash/crc64"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/micro-plat/hydra/conf/server/nfs"
	"github.com/micro-plat/lib4go/errs"
)

//module 协调本地文件、本地指纹、远程指纹等处理
type module struct {
	c        *nfs.NFS
	local    *local
	remoting *remoting
	async    *async
	once     sync.Once
}

func newModule(c *nfs.NFS) (m *module) {
	m = &module{
		local:    newLocal(c.Local),
		remoting: newRemoting(),
	}
	m.async = newAsync(m.local, m.remoting)
	return m
}

//Update 更新环境配置
func (m *module) Update(hosts []string, masterHost string, currentAddr string, isMaster bool) {
	m.remoting.Update(hosts, masterHost, currentAddr, isMaster)
	m.local.Update(currentAddr)
	if isMaster {
		m.async.DoQuery()
	}
}

//checkAndDownload 判断整个集群是否存在文件
//1. 检查本地是否存在
//2. 从master获取指纹信息，看哪些服务器有此文件
//3. 向有此文件的服务器拉取文件
//4. 保存到本地
//5. 通知master我也有这个文件了,如果是master则告诉所有人我也有此文件了
func (m *module) checkAndDownload(name string) error {

	//从本地获取
	if m.local.Has(name) {
		return nil
	}

	//从远程获取
	fp, err := m.remoting.GetFP(name)
	if err != nil {
		return err
	}

	//从远程拉取文件
	buff, err := m.remoting.Pull(fp)
	if err != nil {
		return err
	}

	//保存到本地
	fp, err = m.local.SaveFile(name, buff, fp.Hosts...)
	if err != nil {
		return err
	}

	//上报给其它服务器
	m.async.DoReport(fp.GetMAP())
	return nil
}

//HasFile 本地是否存在文件
func (m *module) HasFile(name string) error {
	if m.local.Has(name) {
		return nil
	}
	return errs.NewErrorf(http.StatusNotFound, "文件%s不存在", name)
}

//SaveNewFile 保存新文件到本地
//1. 查询本地是否有此文件了,有则报错
//2. 保存到本地，返回指纹信息
//3. 通知master我也有这个文件了,如果是master则告诉所有人我也有此文件了
func (m *module) SaveNewFile(name string, buff []byte) (*eFileFP, error) {
	//检查文件是否存在
	name = getFileName(name)
	if m.local.Has(name) {
		return nil, fmt.Errorf("文件名称重复:%s", name)
	}

	//保存到本地
	fp, err := m.local.SaveFile(name, buff)
	if err != nil {
		return nil, err
	}

	//远程通知
	m.async.DoReport(fp.GetMAP())
	return fp, nil
}

//GetFP 获取本地的指纹信息，用于master对外提供服务
//1. 查询本地是否有文件的指纹信息
//2. 如果是master返回不存在
//3. 向master发起查询
func (m *module) GetFP(name string) (*eFileFP, error) {
	//从本地文件获取
	if f, ok := m.local.GetFP(name); ok {
		return f, nil
	}
	return nil, errs.NewError(http.StatusNotFound, "文件不存在")
}

//Query 获取本地包含有服务列表的指纹清单
func (m *module) Query() eFileFPLists {
	return m.local.GetFPs()
}

//RecvNotify 接收远程发过来的新文件通知
//1. 检查本地是否有些文件
//2. 文件不存在则自动下载
//3. 合并服务列表
func (m *module) RecvNotify(f eFileFPLists) error {
	//处理本地新文件上报
	reports, downloads, err := m.local.Merge(f)
	if err != nil {
		return err
	}

	//上报到服务
	m.async.DoReport(reports)
	for _, f := range downloads {
		m.async.DoDownload(f)
	}
	return m.local.FPWrite(m.local.FPS.Items())
}

//Close 关闭服务
func (m *module) Close() error {
	m.once.Do(func() {
		m.local.Close()
		m.async.Close()
	})
	return nil
}

func getCRC64(buff []byte) uint64 {
	return crc64.Checksum(buff, crc64.MakeTable(crc64.ISO))
}

func getFileName(name string) string {
	return filepath.Join(time.Now().Format("20060102"), name)
}
