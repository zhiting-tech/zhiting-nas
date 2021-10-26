package task

import (
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"strconv"
	"sync"
	"time"
)

type DetailInterface interface {
	ExecTask() error
}

// Detail 任务详情
type Detail struct {
	Topic  string // 任务类型
	DetailInterface DetailInterface
	Sign string // 标识，对应topic下的唯一标识
	Status int // 状态 1未开始｜2开始中｜3失败
}

var taskManager *Manager
var _once sync.Once

func GetTaskManager() *Manager {
	_once.Do(func() {
		taskManager = &Manager {
			m: sync.Mutex{},
			Tasks: make(map[string]*Detail),
		}
	})
	return taskManager
}

// Manager 任务管理器
type Manager struct {
	m		   sync.Mutex
	Tasks      map[string]*Detail  // tasks
	TaskSlice  []string
}

// Add 添加任务
func (manager *Manager) Add(topic string, sign string, detailInterface DetailInterface)  {
	manager.m.Lock()
	defer manager.m.Unlock()

	if sign == "" {
		// 如果标示为空，则默认取md5(时间戳)
		sign = utils.EncodeMD5(strconv.FormatInt(time.Now().UnixNano(), 10))
	}

	key := fmt.Sprintf("%s_%s", topic, sign)

	manager.Tasks[key] = &Detail{
		Topic:  topic,
		Sign: sign,
		DetailInterface: detailInterface,
		Status: types.TaskOnGoing,
	}
	manager.TaskSlice = append(manager.TaskSlice, key)
}

func (manager *Manager) Start() {
	for {
		time.Sleep(1 * time.Second)
		if len(manager.TaskSlice) == 0 {
			continue
		}
		key := manager.PopSlice()
		task, ok := manager.Tasks[key]
		if !ok {
			// 如果没有该任务，则不往下执行
			continue
		}
		if task.Status == types.TaskFailed {
			// 如果任务执行失败，则重新放进任务里，等待重新执行
			manager.PushSlice(key)
			continue
		}

		// 设置任务为正在执行
		manager.Tasks[key].Status = types.TaskOnGoing
		if err := task.DetailInterface.ExecTask(); err != nil {
			config.Logger.Errorf("%s 的错误为 %v", key, err)
			// 设置任务为失败， 重新放入任务队列
			manager.Tasks[key].Status = types.TaskFailed
			manager.PushSlice(key)
		} else {
			// 任务执行成功
			config.Logger.Infof("%s执行完毕", key)
			delete(manager.Tasks, key)
		}
	}
}

// GetTaskInfo 返回任务详情
func (manager *Manager) GetTaskInfo(topic string, sign string) (*Detail, bool){
	key := fmt.Sprintf("%s_%s", topic, sign)
	task, ok := manager.Tasks[key]
	return task, ok
}

// GetTaskInfoByKey 根据map的key获取任务详情
func (manager *Manager) GetTaskInfoByKey(key string) (*Detail, bool) {
	task, ok := manager.Tasks[key]
	return task, ok
}

// DelByKey 删除任务
func (manager *Manager) DelByKey(key string) {
	delete(manager.Tasks, key)
}

// RestartByKey 重新开始某个任务任务
func (manager *Manager) RestartByKey(key string) {
	manager.Tasks[key].Status = types.TaskOnGoing
}

// PopSlice 弹出一个任务key
func (manager *Manager) PopSlice() string {
	manager.m.Lock()
	defer manager.m.Unlock()

	key := manager.TaskSlice[0]
	manager.TaskSlice = manager.TaskSlice[1:]

	return key
}

// PushSlice 把任务key放入slice里
func (manager *Manager) PushSlice(key string) {
	manager.m.Lock()
	defer manager.m.Unlock()

	manager.TaskSlice = append(manager.TaskSlice, key)
}