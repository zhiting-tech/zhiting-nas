package task_test

import (
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"testing"
	"time"
)

type TestTaskStruct struct {
	num int
}

func (t *TestTaskStruct) ExecTask() error  {
	fmt.Println(t.num)
	return nil
}

func TestTask(t *testing.T) {
	manager := task.GetTaskManager()

	for i := 0; i < 10; i++ {
		tmp := &TestTaskStruct{num: i}
		manager.Add("good", fmt.Sprintf("%d", i), tmp)
	}

	go manager.Start()

	time.Sleep(100 * time.Second)
}