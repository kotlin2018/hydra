package uuid

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/micro-plat/hydra/context"
	"github.com/micro-plat/hydra/global"
	"github.com/micro-plat/hydra/registry"
)

// 因为UUID目的是解决分布式下生成唯一id 所以ID中是包含集群和节点编号在内的

const (
	workerBits uint8 = 10 // 每台机器(节点)的ID位数 10位最大可以有2^10=1024个节点
	numberBits uint8 = 12 // 表示每个集群下的每个节点，1毫秒内可生成的id序号的二进制位数 即每毫秒可生成 2^12-1=4096个唯一ID
	// 这里求最大值使用了位运算，-1 的二进制表示为 1 的补码，感兴趣的同学可以自己算算试试 -1 ^ (-1 << nodeBits) 这里是不是等于 1023
	workerMax   int64 = -1 ^ (-1 << workerBits) // 节点ID的最大值，用于防止溢出
	numberMax   int64 = -1 ^ (-1 << numberBits) // 同上，用来表示生成id序号的最大值
	timeShift   uint8 = workerBits + numberBits // 时间戳向左的偏移量
	workerShift uint8 = numberBits              // 节点ID向左的偏移量
	// 41位字节作为时间戳数值的话 大约68年就会用完
	// 假如你2010年1月1日开始开发系统 如果不减去2010年1月1日的时间戳 那么白白浪费40年的时间戳啊！
	// 这个一旦定义且开始生成ID后千万不要改了 不然可能会生成相同的ID
	//epoch int64 = 1525705533000 // 这个是我在写epoch这个变量时的时间戳(毫秒)
	epoch int64 = 1577808000000 //2020-01-01 0:0:0
)

//UUID 定义一个woker工作节点所需要的基本参数
type UUID struct {
	name      string
	root      string
	registry  registry.IRegistry
	mu        sync.Mutex // 添加互斥锁 确保并发安全
	timestamp int64      // 记录时间戳
	workerID  int64      // 该节点的ID
	number    int64      // 当前毫秒已经生成的id序列号(从0开始累加) 1毫秒内最多生成4096个ID
	closeChan chan struct{}
	done      bool
}

var uuid *UUID
var once sync.Once

//Get 根据当前应用环境构建
func Get() *UUID {
	once.Do(func() {
		r, err := registry.NewRegistry(global.Def.RegistryAddr, global.Def.Log())
		if err != nil {
			panic(err)
		}
		uuid, err = NewUUID(context.Current().ServerConf().GetMainConf().GetServerPubPath(),
			context.Current().ServerConf().GetMainConf().GetClusterID(), r)
		if err != nil {
			panic(err)
		}

	})
	return uuid
}

//NewUUID  实例化一个工作节点
func NewUUID(root string, name string, r registry.IRegistry) (*UUID, error) {

	// 生成一个新节点
	w := &UUID{
		root:      root,
		name:      name,
		registry:  r,
		timestamp: 0,
		number:    0,
		closeChan: make(chan struct{}),
	}
	if err := w.watch(); err != nil {
		return nil, err
	}
	if w.workerID < 0 {
		w.workerID = w.workerID * -1
	}
	if w.workerID > workerMax {
		w.workerID = w.workerID % workerMax
	}
	return w, nil
}

//GetString 获取字符串编号
func (w *UUID) GetString(pre ...string) string {
	return fmt.Sprintf("%s%d", strings.Join(pre, ""), w.Get())
}

//Get 接下来我们开始生成id
// 生成方法一定要挂载在某个woker下，这样逻辑会比较清晰 指定某个节点生成id
func (w *UUID) Get() int64 {
	// 获取id最关键的一点 加锁 加锁 加锁
	w.mu.Lock()
	defer w.mu.Unlock() // 生成完成后记得 解锁 解锁 解锁

	// 获取生成时的时间戳
	now := time.Now().UnixNano() / 1e6 // 纳秒转毫秒
	if w.timestamp == now {
		w.number++

		// 这里要判断，当前工作节点是否在1毫秒内已经生成numberMax个ID
		if w.number > numberMax {
			// 如果当前工作节点在1毫秒内生成的ID已经超过上限 需要等待1毫秒再继续生成
			for now <= w.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		// 如果当前时间与工作节点上一次生成ID的时间不一致 则需要重置工作节点生成ID的序号
		w.number = 0
		w.timestamp = now // 将机器上一次生成ID的时间更新为当前时间
	}

	// 第一段 now - epoch 为该算法目前已经奔跑了xxx毫秒
	// 如果在程序跑了一段时间修改了epoch这个值 可能会导致生成相同的ID
	ID := int64((now-epoch)<<timeShift | (w.workerID << workerShift) | (w.number))
	return ID
}

//watch 监测集群变化
func (w *UUID) watch() (err error) {
	cldrs, _, err := w.registry.GetChildren(w.root)
	if err != nil {
		return err
	}
	w.workerID = getIndex(cldrs, w.name)

	//监控子节点变化
	ch, err := w.registry.WatchChildren(w.root)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-w.closeChan:
				return
			case cldWatcher := <-ch:
				if cldWatcher.GetError() == nil {
					cldrs, _, _ := w.registry.GetChildren(w.root)
					w.workerID = getIndex(cldrs, w.name)
				}
			LOOP:
				ch, err = w.registry.WatchChildren(w.root)
				if err != nil {
					if w.done {
						return
					}
					time.Sleep(time.Second)
					goto LOOP
				}
			}
		}
	}()
	return nil
}
func getIndex(children []string, name string) int64 {
	for i, v := range children {
		if strings.Contains(v, name) {
			return int64(i)
		}
	}
	return 0
}
