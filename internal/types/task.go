package types

const (
	TaskMovingFolder    = "TaskMovingFolder"    // 文件夹移动分区
	TaskDelFolder       = "TaskDelFolder"       // 删除文件夹
	TaskAddPartition    = "TaskAddPartition"    // 添加存储池分区
	TaskUpdatePartition = "TaskUpdatePartition" // 修改存储池分区
	TaskDelPartition    = "TaskDelPartition"    // 删除存储池分区
	TaskDelPool         = "TaskDelPool"         // 删除存储池

	TaskFailed     = 0 // 任务失败
	TaskOnGoing    = 1 // 任务开始中
	TaskNotStarted = 2 // 任务未开始
)
