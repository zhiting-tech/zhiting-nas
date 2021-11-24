

# 网盘插件

​	网盘插件是智汀家庭云（SmartAssistant，以下简称SA）为用户及其家庭提供存储的数据存储中心。网盘插件是以SA插件的形式，通过SA的授权登陆网盘。

## 目录结构

```
├── app.yaml			运行时配置文件
├── app.yaml.example	配置文件范例
├── Dockerfile			Docker相关打包脚本
├── internal
│ ├── api				接口
│ │ ├── disk
│ │ ├── folder
│ │ ├── middleware
│ │ ├── partition
│ │ ├── pool
│ │ ├── resource
│ │ ├── setting
│ │ ├── share
│ │ ├── task
│ │ └── utils
│ ├── config							
│ ├── entity
│ ├── task
│ └── types
│     ├── status
├── main.go				入口文件
├── Makefile			make 配置
├── pkg					通用组件代码
│ ├── errors
│ ├── filebrowser
│ ├── logger
│ ├── response
│ ├── session
│ └── utils
└── README.md			项目介绍文档
```

## 开发环境搭建

网盘插件的使用需要配合智汀家庭云(SA)和LVM服务的支持，通过智汀家庭云(SA)授权登录，通过LVM服务进行io操作。

### 环境准备

+ go 版本为1.15.0或以上
+ 确保LVM服务和智汀家庭云服务能够正常访问

#### 步骤

获取代码

```
git clone https://github.com/zhiting-tech/wangpan.git
```

同步依赖

```
go mod tidy
```

复制 app.yaml.example 到 app.yaml 并配置

```
Server:
    RunMode: debug
    HttpPort: 8089
    ReadTimeout: 
    WriteTimeout: 
App:
    DefaultPageSize: 
    MaxPageSize: 
    LogSavePath: 
    LogFileName: 
    LogFileExt: 
    # 上传文件保存路径
    UploadSavePath: ""
    # SQLite 文件路径
    DbSavePath: ""
ExtServer:
	# LVM服务的访问地址
    LvmServer: 
    # 智汀家庭云的访问地址
    SaServer: ""
    SaHttp: http
```

编译运行

```
go run main.go
```

## 数据库设计

网盘作为一个插件服务使用，对数据量的要求相对来说不高，使用内嵌的SQLite数据库文件存储数据。

### 数据库表

文件表

```sql
create table folder
(
    id				int not null auto_increment,
    u_id			int comment '用户id'
    abs_path		varchar(255) comment '绝对路径'
    name			varchar(255) comment '文件/文件夹名称'
    mode			int(1) comment '文件夹类型：1私人文件夹 2共享文件夹'
    type			int(1) comment '类型：0文件夹 1文件'
    is_encrypt		int(1) comment '是否加密'
    cipher			varchar(255) comment '加密后的密匙'
    pool_name		varchar(255) comment '存储此Id'
    partition_name	varchar(255) comment '存储池分区Id'
    persons			varchar(255) comment '可访问成员，冗余用'
    created_at		int comment '创建时间'
    primary key(id)
);
```

文件权限表

```sql
create table folder_auth
(
	id				int not null auto_increment,
	nickname		varchar(255) comment '用户名称'
	face			varchar(255) comment '头像'
	uid				int comment '用户Id'
	from_user		varchar(255) comment '来源用户'
	is_share		int(1) comment '对于uid用户是否为共享文件'
	folder_id		int comment '文件夹Id'
	read			int(1) comment '是否可读'
	write			int(1) comment '是否可写'
	deleted			int(1) comment '是否可删除'
	primary key(id)
);
```

默认配置表

```sql
create table setting
(
	name			varchar(255) 
	value			varchar(255)
);
```

# 网盘的基本介绍

## 整体架构

网盘使用gin框架作为服务响应的处理，使用LVM管理机制（以下简称LVM）对磁盘进行管理，通过gRPC与LVM进行通信。在实际使用中，部分操作会使用异步任务而非同步任务。异步任务主要面对一些处理时间上相对较长的，将任务置于后台处理，快速响应，处理完成返回结果。用户可以不用等待任务完成而去处理其它事情。

## 初步了解

网盘插件有存储池和存储分区，以及异步任务等设计概念，异步任务会在稍后详细介绍，这里介绍下存储池和存储分区两个概念，有助于后面的理解

### 存储池

存储池是网盘的基本的空间容量。通过LVM将一块硬盘划分为一个有存储容量的分区，一块硬盘的容量决定该存储池的容量。

存储池是网盘的分区概念。只有在建立存储池之后，才能在该存储池中建立存储分区，划分存储分区的容量。

存储池可以理解为windows里的分卷。但windows一个硬盘可以做多个分卷，网盘插件内一个硬盘则只能做一个分卷。

### 存储分区

存储分区是作为存储池下的容量再次划分。存储分区可以自定义存储分区的大小，可以在存储分区内建立文件夹以存储文件。

存储分区在网盘里负责文件夹管理及文件的存储。用户可以浏览文件夹内的文件，亦或是在存储分区的文件夹内将文件上传/下载。

# 网盘开发

## 用户的首次登录

网盘依靠SA的scope-token授权进行登录。

用户首次登录时，会初始化一个以用户昵称为名称的私人文件夹。

### TOKEN检验

sa记录着网盘的授权信息，网盘通过携带用户Id和token向sa发起http请求校验授权信息。校验通过返回状态码0。

```go
apiUrl := fmt.Sprint( "/api/users/", uid)
userInfo, err := utils.GetRequestSaServer(apiUrl, c)
// 判断响应的状态码 为0则http请求成功
if userInfo.Status != 0 {
	……
}
```

### 权限检验

以下代码会根据SA反向代理的HTTP头信息获取用户数据：

```go
func Get(c *gin.Context) *User {
	// 从上下文中获取session_user
	user, exists := c.Get("session_user")
	if !exists {
		return nil
	}
	return user.(*User)
}
```

#### 拥有者权限

拥有者权限可以创建存储池及存储分区，并对其进行添加、编辑、删除等操作。其权限是至关重要的，所以在进行存储池以及存储分区的增删改时，需要检验权限才能确保数据不是因为非用户操作而删除或丢失。根据SA返获取的用户数据，来进行判断：

```go
u := session.Get(c)
if u == nil {
    response.HandleResponse(c, errors.New(status.PoolIsNotPermission), nil)
    c.Abort()
    return
}
if !u.IsOwner {
    response.HandleResponse(c, errors.New(status.PoolIsNotPermission), nil)
    c.Abort()
    return
}
```

#### 读写权限

在用户（包括拥有者本人以外的所有用户）进行目录读取/创建、文件上传/下载/重命名时，需要判断该用户是否拥有权限来进行这些操作。

```go
auth, err := utils.GetFilePathAuth(u.UserID, path)
if err != nil || auth == nil {
    response.HandleResponse(c, errors.New(status.ResourceNotAuthErr), nil)
    c.Abort()
    return
}
// 没有只读权限
if auth.Read == 0 {
    response.HandleResponse(c, errors.New(status.ResourceNotReadAuthErr), nil)
    c.Abort()
    return
}
```

## 异步任务的实现

异步任务是为了在做一些运行过长的操作时，能够将任务转至后台，同时返回响应，在前端告知用户已在执行任务，用户不用等待任务的同步完成，可以在异步任务进行的同时进行一些其他的浏览或操作。

异步任务的实现是利用一个切片，在程序初始化时，启动一个协程来循环扫描这个切片。当切片中加入一个或多个任务时，会取出一个任务执行。并且会有辨别码，辨别任务是否是执行中/执行完毕/执行失败。

```go
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
```

### 异步任务的添加

将任务的信息写入map，再将该map写入到任务切片中等待执行：

```go
func (manager *Manager) Add(topic string, sign string, detailInterface DetailInterface)  {
	……
    key := fmt.Sprintf("%s_%s", topic, sign)
    manager.Tasks[key] = &Detail{
        Topic:  topic,
        Sign: sign,
        DetailInterface: detailInterface,
        Status: types.TaskOnGoing,
    }
    manager.TaskSlice = append(manager.TaskSlice, key)
}
```

### 异步任务的删除

异步任务删除需要根据key从切片中获取其状态，判断该状态是否可以在切片删除：

```go
taskInfo, exist := task.GetTaskManager().GetTaskInfoByKey(req.Id)
if taskInfo.Status == types.TaskOnGoing {
	err = errors.Wrap(err, status.TaskStatusErr)
	return
}
task.GetTaskManager().DelByKey(req.Id)

func (manager *Manager) DelByKey(key string) {
	delete(manager.Tasks, key)
}
```

### 异步任务的重新开始

异步任务重新开始需要根据key从切片中获取状态，判断状态是否可以重新加入切片中

```go
taskInfo, exist := task.GetTaskManager().GetTaskInfoByKey(req.Id)
if taskInfo.Status == types.TaskOnGoing {
	err = errors.Wrap(err, status.TaskStatusErr)
	return
}
task.GetTaskManager().RestartByKey(req.Id)

func (manager *Manager) RestartByKey(key string) {
	manager.Tasks[key].Status = types.TaskOnGoing
}
```

## 物理分区管理

物理分区的管理需要通过GPRC + Protobuf 调用LVM的服务进行操作，在每一次操作前都需要进行GRPC连接，后文不再赘述，默认连接已建立

gRPC连接：

```
conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
if err != nil {
	……
}
defer conn.Close()
client := proto.NewDiskManagerClient(conn)
```

### 添加物理分区到存储池

填充指定结构体，生成请求，并添加物理分区到存储池：

```go
ctx := context.Background()

createReq := proto.VolumeGroupCreateOrExtendReq{
	VGName: req.PoolName,
	PVName: req.DiskName,
}
result, err := client.VolumeGroupExtend(ctx, &createReq)
```

### 获取物理分区列表

获取物理分区列表：

```go
VList, err := client.PhysicalVolumeList(ctx, &proto.Empty{})
```

解析并获取VList内的数据，并填入至变量中响应客户端请求：

```go
for _, pv := range VList.PVS {
	if pv.VGName == "" {
		info := &ListResp{
			Id:          pv.UUID,
			Name:        pv.Name,
			VGName:      pv.VGName,
			Capacity:    pv.Size,
		}
		list = append(list, info)
	}
}
totalRow = int64(len(list))
response.HandleResponseList(c, err, &list, totalRow)
```

## 存储池管理

存储此的管理同样也需要调用LVM服务，有关GRPC连接的内容不再赘述

### 存储池的添加

填写结构体变量，请求LVM创建存储池：

```go
createReq := proto.VolumeGroupCreateOrExtendReq{
    VGName: req.Name,
    PVName: req.DiskName,
}
result, err := client.VolumeGroupCreate(ctx, &createReq)
```

### 存储池的删除

​	填写结构体变量，请求LVM删除存储池：

```go
param := proto.VolumeGroupRemoveReq{
	VGName: req.Name,
}
result, err := client.VolumeGroupRemove(ctx, &param)
```

### 存储池的更新

填写结构体变量，请求LVM更新存储池名称：

```go
updateReq := proto.VolumeGroupRenameReq{
	OldName: req.Name,
	NewName: req.NewName,
}
result, err := client.VolumeGroupRename(ctx, &updateReq)
```

更新数据库中的路径名称：

```go
oldPath := fmt.Sprintf("/%s", req.Name)
newPath := fmt.Sprintf("/%s", req.NewName)
if err = utils.UpdateFolderPath(entity.GetDB(), oldPath, newPath);
```

如果更新的存储池为默认设置中的存储池，则需要更新数据库以及全局：

```go
if req.Name == config.AppSetting.PoolName {
	entity.UpdatePoolNameSetting(req.NewName)
	config.AppSetting.PoolName = req.NewName
}
```

### 获取存储池的信息

请求LVM返回所有存储池信息以及正在进行的存储池异步任务，并匹配：

```go
taskListMap := getPartitionAddTask()
groups, err := client.VolumeGroupList(ctx, &proto.Empty{})
for _, vg := range groups.VGS {
	if vg.Name != req.Name {
		// 使用名称进行匹配
		continue
	}
}
```

匹配完毕，需要获取存储池及其下的存储分区的异步任务信息，并将物理分区信息写入（赋值部分已省略）。

```go
for key, lv := range vg.LVS {
	taskId, status := getPartitionTaskInfo(req.Name, lv.Name)
	resp.Lv[key] = &LogicalVolume{
        ……
   	}
}
if lvAddTaskList, ok := taskListMap[vg.Name]; ok {
	for _, value := range lvAddTaskList {
		if _, ok = lvMap[value.Name]; !ok {
			resp.Lv = append(resp.Lv, value)
		}
	}
}
resp.Pv = make([]*PhysicalVolume, len(vg.PVS))
for key, pv := range vg.PVS {
		……
}
```

### 获取所有存储池的列表

请求LVM获取所有的存储池信息：

```
groups, err := client.VolumeGroupList(ctx, &proto.Empty{})
```

获取存储池的任务状态并赋值：

```
for _, vg := range groups.VGS {
	taskId, status := getPoolTaskInfo(vg.Name)
	info := &ListResp{
		……
	}
}
```

逻辑分区赋值：

```
info.Lv = make([]*LogicalVolume, len(vg.LVS))
for key, lv := range vg.LVS {
	info.Lv[key] = &LogicalVolume{
		……
	}
}
```

物理分区赋值，并写入指定切片中返回：

```
info.Pv = make([]*PhysicalVolume, len(vg.PVS))
for key, pv := range vg.PVS {
	info.Pv[key] = &PhysicalVolume{
		……
	}
	list = append(list, info)
}
```

## 存储分区管理

存储分区的管理需要异步任务的执行。涉及GRPC连接以及请求内容是否合法等内容不再赘述

### 存储分区的添加

在异步任务中，请求LVM创建新存储分区：

```go
param := proto.LogicalVolumeCreateReq{
	VGName: req.PoolName,
	LVName: req.Name,
	SizeM:  req.Capacity,
}
result, err := client.LogicalVolumeCreate(ctx, &param)
```

### 存储分区的删除

在异步任务中，首先是请求LVM删除存储分区：

```go
delReq := proto.LogicalVolumeRemoveReq{
	VGName: req.PoolName,
	LVName: req.Name,
}
result, err := client.LogicalVolumeRemove(ctx, &delReq)
```

并且在数据库中把对应的文件夹信息删除：

```go
absPath := fmt.Sprintf("/%s/%s", req.PoolName, req.Name)
_ = entity.DelFolder(entity.GetDB(), absPath)
```

如果删除的存储池是配置中的存储池，则将配置中的存储分区及存储池改成默认的存储分区和存储池。

### 存储分区的重命名和扩容

存储分区的重命名是同步任务，扩容是异步任务。

#### 重命名

首先是请求LVM重命名存储分区：

```go
param := proto.LogicalVolumeRenameReq{
	VGName:    req.PoolName,
	LVName:    req.Name,
	NewLVName: req.NewName,
}
result, err := client.LogicalVolumeRename(ctx, &param)
```

需要更新数据库内的文件夹信息：

```go
oldPath := fmt.Sprintf("/%s/%s", req.PoolName, req.Name)
newPath := fmt.Sprintf("/%s/%s", req.PoolName, req.NewName)
utils.UpdateFolderPath(entity.GetDB(), oldPath, newPath);
```

如果修改的是配置项的存储池名称，还需要修改对应的设置以及更新数据库设置。

```go
if config.AppSetting.PoolName == req.PoolName && config.AppSetting.PartitionName == req.Name {
	entity.UpdatePartitionNameSetting(req.NewName)
	config.AppSetting.PartitionName = req.NewName
}
```

#### 扩容

在异步任务中，请求LVM扩容存储分区：

```go
param := proto.LogicalVolumeExtendReq{
	VGName:   req.PoolName,
	LVName:   req.NewName,
	NewSizeM: req.Capacity,
}
result, err := client.LogicalVolumeExtend(ctx, &param)
```

## 文件夹管理

用户对文件夹进行添加、删除、重命名等基础操作的同时，还可以对文件夹是否进行加密，是否共享，其他用户是否有权限访问私人文件夹等进行操作。

文件夹的操作均通过调用封装好的GetFB()方法实现，以下用FB简称代替。

```go
func GetFB() *FileBrowser {
	once.Do(func() {
		fb = &FileBrowser{
			fs:       nil,
			dirMode:  0777,
			fileMode: 0666,
		}
		rootPath := config.AppSetting.UploadSavePath
		if !path.IsAbs(rootPath) {
			wd, err := os.Getwd()
			if err != nil {
				log.Fatalf("can not read current dir, error: %v", err.Error())
			}
			rootPath = filepath.Join(wd, rootPath)
		}
		log.Printf("use %v as file root path", rootPath)

		if err := os.MkdirAll(rootPath, fb.dirMode); err != nil {
			log.Fatalf("can not create root data dir, error: %v", err.Error())
		}
		fb.root = rootPath
		fb.fs = afero.NewBasePathFs(afero.NewOsFs(), rootPath)

	})
	return fb
}
```

文件夹在数据库中分为两个部分：记录文件夹信息的folder表，以及记录文件夹的用户权限信息的folder_auth表。

### 文件夹的添加

获取到请求时，将请求中的数据分别写入用户权限信息结构体和文件夹信息结构体：

```go
for key, auth := range req.Auth {
	auths[key] = entity.FolderAuth{
		Uid:      auth.Uid,
		Nickname: auth.Nickname,
		Face:     auth.Face,
		Read:     auth.Read,
		Write:    auth.Write,
		Deleted:  auth.Deleted,
	}
	persons[key] = auth.Nickname
}
```

通过FB创建文件夹，并将数据填充结构体，写入folder表：

```go
err = filebrowser.GetFB().Mkdir(fmt.Sprintf("/%s/%s/%s", req.PoolName, req.PartitionName, req.Name))
folderInfo, err := entity.CreateFolder(tx, &entity.FolderInfo{
			Name:          req.Name,
			Uid:           user.UserID,
			Mode:          req.Mode,
			PoolName:      req.PoolName,
			PartitionName: req.PartitionName,
			IsEncrypt:     req.IsEncrypt,
			Cipher:        req.Cipher,
			Type:          types.FolderTypeDir,
			CreatedAt:     time.Now().Unix(),
			Persons:       strings.Join(persons, "、"),
			AbsPath:       fmt.Sprintf("/%s/%s/%s", req.PoolName, req.PartitionName, req.Name),
		})
```

判断文件夹是否是共享文件夹，并将用户权限信息填写完整，写入folder_auth表：

```go
isShare := 0
if req.Mode == types.FolderShareDir {
	isShare = 1
}
for key := range req.Auth {
	auths[key].FolderId = folderInfo.ID
	auths[key].IsShare = isShare
}
if err = entity.BatchInsertAuth(tx, auths); err != nil {}
```

### 文件夹的更新

文件夹的更新也分为两个部分，更新文件夹的名字、类型（私人or共享），有权限的用户名、以及更新用户权限信息。

更新文件夹信息：

```go
values := map[string]interface{}{
	"name":           req.Name,
	"mode":           req.Mode,
	"Persons":        strings.Join(persons, "、"), // 可访问成员
}
if err = entity.UpdateFolderInfo(tx, req.ID, values); err != nil {}
```

更新用户权限信息，需要先删除权限，在添加权限信息：

```go
if err = entity.DelFolderAuth(tx, req.ID); err != nil{}
isShare := 0
if req.Mode == types.FolderShareDir {
	isShare = 1
}
for key := range auths {
	auths[key].FolderId = req.ID
	auths[key].IsShare = isShare
}
if err = entity.BatchInsertAuth(tx, auths); err != nil {}
```

### 文件夹的删除

删除文件夹时，需要删除folder表、folder_auth表中的相关信息，最后通过FB删除：

```go
if err := entity.DelFolder(tx, oldInfo.AbsPath); err != nil {
	return errors.Wrap(err, status.FolderDelFailErr)
}
if err = entity.DelFolderAuth(tx, req.Id); err != nil {
	return errors.Wrap(err, status.FolderDelFailErr)
}
if err = filebrowser.GetFB().RemoveAll(oldInfo.AbsPath); err != nil {
	return errors.Wrap(err, status.FolderDelFailErr)
}
```

### 文件夹的解密和修改密码

文件夹的解密和重新加密都用到了golang的crypro标准库的aes包和cipher包。

查询数据库获取密钥：

```go
folderInfo, err := entity.GetFolderInfo(req.Id)
```

将旧密码还原：

```go
secret, err := utils.DecryptString(req.OldPwd, folderInfo.Cipher)
```

加密新密码，并更新数据库：

```go
cipher, err := utils.EncryptString(req.NewPwd, secret)
if err = entity.UpdateFolderInfo(entity.GetDB(), req.Id, entity.FolderInfo{Cipher: cipher});
```

文件夹解除密码时只需要调用utils内函数即可：

```go
_, err = utils.GetFolderSecret(req.Path, req.Password)
```

### 移除用户时，需要删除那些文件夹

在网盘里移除用户时，我们需要查找其建立的私人文件夹：

```go
folderInfos, err := entity.GetPrivateFolders(req.UserIDs)
for _, folderInfo := range folderInfos {
	err = removeFolderAndRecode(fs, folderInfo.AbsPath)
	if err != nil {
        return
	}
}
```

并且要删除其初始化的文件夹：

```go
for _, v := range req.UserIDs {
	folderRow, err := entity.GetRelateFolderInfoByUid(types.FolderSelfDirUid, v)
	if err != nil {
		return
	}
	err = removeFolderAndRecode(fs, folderRow.AbsPath)
	if err != nil {
		return
	}
}
```

最后要删除folder_auth表中关于该用户UID的所有记录：

```go
if err = entity.DelFolderAuthByUid(req.UserIDs); err != nil {
	return
}
```

### 获取文件夹列表

通过查询数据库获取所有的文件夹信息并获取文件夹的异步任务信息：

```go
pageOffset := utils.GetPageOffset(req.Page, req.PageSize)
folderInfos, err := entity.GetFolderList(user.UserID, pageOffset, req.PageSize)
if err != nil {
	return
}

for _, folderInfo := range folderInfos {
	taskId, status := getFolderTaskInfo(folderInfo.ID)
	list = append(list, &Info{
		ID:        folderInfo.ID,
		Name:      folderInfo.Name,
		IsEncrypt: folderInfo.IsEncrypt,
		Mode:      folderInfo.Mode,
		Path:      folderInfo.AbsPath,
		Type:      folderInfo.Type,
		Persons:   folderInfo.Persons,
		PoolName:  fmt.Sprintf("%s-%s", folderInfo.PoolName, folderInfo.PartitionName),
		Status:	   status,
		TaskId:	   taskId,
	})
}
```

### 获取文件夹的信息

文件夹的信息分为两个部分，一个部分是文件夹的本体信息，另一个部分是该文件夹的用户权限信息：

```go
info, err := entity.GetFolderInfo(req.Id)
folderAuthList, err := entity.GetFolderAuthByFolderId(req.Id)

resp.ID = info.ID
resp.Name = info.Name
resp.IsEncrypt = info.IsEncrypt
resp.Mode = info.Mode
resp.Type = info.Type
resp.PoolName = info.PoolName
resp.PartitionName = info.PartitionName

for _, auth := range folderAuthList {
	resp.Auth = append(resp.Auth, AddAuthResp{
		Uid:      auth.Uid,
		Nickname: auth.Nickname,
		Face:     auth.Face,
		Read:     auth.Read,
		Write:    auth.Write,
		Deleted:  auth.Deleted,
	})
}
```



## 上传/下载文件及管理

### 文件的上传

文件的上传是以分块的形式上传，文件的哈希值为缓存目录，存储在缓存路径下。当分块上传完毕后，则会合并文件夹，将缓存文件夹删除。

#### 分块上传

文件的分块已经由前端切割好，发送的请求中req.Action字段会赋值为chunk，网盘只用负责接收请求并根据请求中的动作做出相对应的处理。

为保证文件的完整性，会使用文件的哈希值作为缓存路径的文件夹名，并在该路径下生成缓存文件。

```go
cachePath := req.getCachePath(user.UserID)
if err = fs.Mkdir(cachePath); err != nil {
	err = errors.Wrap(err, errors.InternalServerErr)
	return
}

if req.isFileExist(filepath.Join(cachePath, req.chunkNumber)) {
	return
}

tmpFile, err := ioutil.TempFile(filepath.Join(fs.GetRoot(), cachePath), "temp-")
if err != nil {
	err = errors.Wrap(err, errors.InternalServerErr)
	return
}
defer tmpFile.Close()
```

赋值上传文件的内容，重新命名缓存文件。

#### 分块合并

当上传最后一块分块时，req.Action会赋值为merge，插件会根据该字段进行分块合并的操作。

打开缓存路径，统计分块总数是否与请求中的分块总数一致：

```go
fileInfos, err := file.Readdir(-1)
cachePath := req.getCachePath(user.UserID)
file, err := fs.Open(cachePath)

totalChunks, _ := strconv.Atoi(req.TotalChunks)
if len(fileInfos) != totalChunks {
	err = errors.New(status.ChunkFileNotExistErr)
	return
}
```

创建临时文件，将缓存目录下的所有的缓存文件写入到临时文件中，并检验临时文件的哈希值。

```go
tempFile, err := ioutil.TempFile(filepath.Join(fs.GetRoot(), cachePath), "temp-")
if err != nil {
	err = errors.Wrap(err, errors.InternalServerErr)
	return
}

for i := range fileInfos {
	var chunkFile filebrowser.File
	chunkFile, err = fs.Open(filepath.Join(cachePath, strconv.Itoa(i+1)))
	if err != nil {
		return resp, err
	}
	var b []byte
	b, err = ioutil.ReadAll(chunkFile)
	if err != nil {
		return resp, err
	}
	tempFile.Write(b)
	chunkFile.Close()
}
tempFile.Close()

rootPath := strings.TrimPrefix(tempFile.Name(), fs.GetRoot())
if err = req.checkFileHash(rootPath); err != nil {
	return
}
```

移动并重命名临时文件，并对其目标目录判断是否需要密钥，如果需要密钥则要对复制后的文件进行加密处理，如果没有就直接将文件复制到目标目录下，删除临时文件，：

```go
fileName := strings.TrimPrefix(tempFile.Name(), fs.GetRoot())

// 获取目录的密钥且校验密码，如果密钥为空，则不需要加密，
secret, err := utils.GetFolderSecret(req.path, c.GetHeader("pwd"))
if secret != "" {
	err = fs.CopyFileToTarget(fileName, newPath + types.FolderEncryptExt) // 如果需要的话，需要加上.env文件
	if err != nil {
		……
		return
	}
	_, err = utils2.EncryptFile(secret, newPath + types.FolderEncryptExt, newPath)
	if err != nil {
		……
		return
	}
	// 把源文件删除
	_ = fs.Remove(newPath + types.FolderEncryptExt)
} else {
	err = fs.CopyFileToTarget(fileName, newPath)
	if err != nil {
		……
		return
	}
}
// 成功后删除原来的文件
_ = fs.RemoveAll(fileName)

if err = req.createFolder(newPath, types.FolderTypeFile, user.UserID); err != nil {
	err = errors.Wrap(err, errors.InternalServerErr)
	return
}

resp, err = req.wrapResp(newPath, fs)
if err == nil {
	// 如果合并成功，把分片的文件夹删除
	_ = fs.RemoveAll(cachePath)
}
```

### 文件的下载

文件下载需要先判断是否当前用户是有否写权限：

```go
write, _ := c.Get("write")
```

如果文件有密码，则需要先解密：

```go
if secret != "" {
	ext := strconv.FormatInt(time.Now().UnixNano(), 10)
	downloadPath, err = utils2.DecryptFile(pwd, downloadPath, fmt.Sprint(downloadPath, ".", ext))
	if err != nil {
		return
	}
}
```

但没有密码就直接打开原文件：

```go
open, err := fb.Open(downloadPath)
```

最后通过http包中的ServeContent方法传输文件

```go
http.ServeContent(c.Writer, c.Request, fileName, fileInfo.ModTime(), open)
```

### 文件的删除

删除文件时，需要判断是否是目录，如果是目录，则需要通过FB将其下的所有文件都删除，否则就根据路径删除文件：

```go
fileInfo, err = fs.Stat(path)
	if err != nil {
		return
	}

if fileInfo.IsDir() {
	if err = fs.RemoveAll(path); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
} else {
	if err = fs.Remove(path); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
}
```

最后，删除folder表中的关联信息

```go
if err = entity.DelFolderByAbsPaths(entity.GetDB(), req.Paths); err != nil{}
```

### 文件的复制和移动

文件进行复制或移动时，需要检查参数，包括req.Action请求是否合法、路径是否合法、以及本次操作的可行性，并获取文件夹密钥和目标文件夹。

```go
if err = req.validateRequest(user.UserID); err != nil{}
secret, err := utils.GetFolderSecret(req.Destination, req.DestinationPwd)
req.Destination, err = utils.GetNewPath(req.Destination)
```

复制需要FB判断该目标是文件夹还是文件再做复制：

```go
isDir, err := fs.IsDir(path)
if err != nil {
	return
}
if !isDir {
	if err = fs.CopyFile(path, req.Destination); err != nil {
		return
	}
} else {
	if err = fs.CopyDir(path, req.Destination); err != nil {
		return
	}
}
```

移动文件/文件夹同样是需要FB判断目标是文件夹还是文件，之后先将文件/文件夹复制到目标目录后再将原路径下的删除：

```go
isDir, _ := fs.IsDir(path)
if isDir {
	// 如果是目录， 先复制
	if err = fs.CopyDir(path, req.Destination); err != nil {
		config.Logger.Errorf("resource_operate CopyDir err %v", err)
		return
	}
} else {
	// 如果是文件， 先复制
	if err = fs.CopyFile(path, req.Destination); err != nil {
		config.Logger.Errorf("resource_operate MoveFile err %v", err)
		return
	}
}
if err = filebrowser.GetFB().RemoveAll(path); err != nil {
	config.Logger.Errorf("resource_operate RemoveAll err %v", err)
	return
}
	// 更新原文件的数据，调整路径
newPath := filepath.Join(req.Destination, filepath.Base(path))
if err = utils.UpdateFolderPath(entity.GetDB(), path, newPath); err != nil {
	config.Logger.Errorf("resource_operate UpdateFolderPath err %v", err)
	return
}
```

最后需要保存数据，并且判断是否需要做加密解密处理：

```go
destPath := filepath.Join(req.Destination, filepath.Base(path))
if err = req.saveFolder(key, uid, secret, destPath); err != nil {
	config.Logger.Errorf("resource_operate save folder err %v", err)
	return
}
```

### 文件的重命名

文件的重命名比较简单，只需要通过FB重命名，在判断错误信息即可：

```go
if err = fs.Rename(req.Path, newPath); err != nil {
	……
}
```

## 获取文件资源的信息

获取文件信息时需要将路径由/s/转换为实际的路径：

```go
newPath, err := utils.GetNewPath(req.Path)
```

读取数据并返回：

```
list, totalRow, err = req.wrapResources(newPath, c)
```

在req.wrapResources中，如果根目录下没有任何的文件，则为用户第一次登录，需要初始化一个私人文件夹，并将数据读取；如果不是，则直接获取文件夹及文件信息：

```go
if newPath == "" {
	err = initPrivateFolder(user)
	if err != nil {
		return
	}
	// 获取私人文件夹 && 可访问权限包含自己的文件夹 && 非分享文件夹
	whereStr := fmt.Sprintf("auth.uid = %d and auth.read = 1 and folder.mode = 1 and auth.is_share = 0", user.UserID)
	folderList, _ := entity.GetRelateFolderList(whereStr, req.PageOffset, req.PageSize)
	totalRow, _ = entity.GetRelateFolderCount(whereStr)
	
	for _, folder := range folderList {
		infos = append(infos, Info{
			Name:      folder.Name,
			Type:      folder.Type,
			Path:      fmt.Sprintf("/s/%d", folder.Id),
			IsEncrypt: folder.IsEncrypt,
			Read:      folder.Read,
			Write:     folder.Write,
			Deleted:   folder.Deleted,
		})
	}
} else {
	infos, err = req.GetResourceInfos(newPath, c)
	if err != nil {
		return
	}
	totalRow = int64(len(infos))
	// type不为1时处理分页
	if req.Type != GetAllFile {
		req.handlePage(infos)
		infos = infos[req.PageOffset:req.PageSize]
	}
}
```

并且尝试从path中获取对应的folderid，如果获取到了就将路径改成/s/:id

```go
folderId, _ := utils.GetFolderIdFromPath(req.Path)
folderInfo, _ := entity.GetFolderInfo(folderId)
if folderId != 0 {
	for i, rs := range infos {
		// 更换路径， 保留/s/:id, 格式
	infos[i].Path = fmt.Sprintf("/s/%d%s", folderId, strings.TrimPrefix(rs.Path, folderinfo.AbsPath))
	}
}
```

## 用户设置

### 用户设置的读取

用户的设置保存在数据库中的setting表，只需要读取数据库并解析即可：

```go
list, err := entity.GetSettingList()
for _, val := range list {
	switch val.Name {
	case "PoolName":
		resp.PoolName = val.Value
	case "PartitionName":
		resp.PartitionName = val.Value
	case "IsAutoDel":
		resp.IsAutoDel, _ = strconv.Atoi(val.Value)
	}
}
```

### 用户设置的更新

当更新用户设置时，需要先把setting表中的数据清空，再将新设置插入切片中，插入setting表，并更新全局的配置：

```go
if err := entity.DropSetting(tx); err != nil {
	return errors.Wrap(err, status.SettingUpdateFailErr)
}
// 默认3个配置
settings := make([]entity.Setting, 0, 3)
settings = append(settings, entity.Setting{Name: "PoolName", Value: req.PoolName})
settings = append(settings, entity.Setting{Name: "PartitionName", Value: req.PartitionName})
settings = append(settings, entity.Setting{Name: "IsAutoDel", Value: strconv.Itoa(req.IsAutoDel)})
if err := entity.BatchInsertSetting(tx, settings); err != nil {
	return errors.Wrap(err, status.SettingUpdateFailErr)
}
// 更新全局配置
config.AppSetting.PoolName = req.PoolName
config.AppSetting.PartitionName = req.PartitionName
config.AppSetting.IsAutoDel = req.IsAutoDel
```

## 共享文件

### 共享文件夹列表

从数据库中获取别人共享的文件：

```go
whereStr := fmt.Sprintf("auth.uid = %d and auth.read = 1 and auth.is_share = 1", user.UserID)
folderList, err := entity.GetRelateFolderList(whereStr, req.PageOffset, req.PageSize)
```

并将切片内的赋值到返回切片中：

```go
for _, folderRow := range folderList {
	isFamilyPath := 0
	if filepath.Base(folderRow.AbsPath) == types.FolderFamilyDir {
		isFamilyPath = 1
	}
	list = append(list, Info{
		ID:       folderRow.Id,
		Name:     folderRow.Name,
		Path:     fmt.Sprintf("/s/%d", folderRow.Id),
		FromUser: folderRow.FromUser,
		Read:     folderRow.Read,
		Write:    folderRow.Write,
		Deleted:  folderRow.Deleted,
		IsFamilyPath: isFamilyPath,
	})
}
```

### 共享文件资源

共享文件夹设置成分享时，需要将原权限删除，并重新组建一个新的权限数组写入至数据库中。

```
// path转换为实际路径
folderId, err = utils.GetAbsFolderIdFromPath(path)
for _, uID := range req.ToUsers {
	// 权限存在则删除
	err = entity.DelFolderAuthByUidAndFolderId(uID, folderId)
	folderAuthCreate := entity.FolderAuth{
		Uid:      uID,
		FromUser: nickname,
		IsShare:  1,
		FolderId: folderId,
		Read:     req.Read,
		Write:    req.Write,
		Deleted:  req.Deleted,
	}
	folderAuthCreates = append(folderAuthCreates, folderAuthCreate)
}
if folderAuthCreates != nil {
	if err = entity.BatchInsertAuth(entity.GetDB(), folderAuthCreates); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
}
```

## 数据库的存储

网盘使用SQL-LITE为的数据库，GORM库进行数据库链接及操作。

网盘的数据库包含三个表：folder、folder_auth、setting。这三个表分别对应文件夹信息、文件夹的用户权限信息、用户设置。每个表都有编写丰富的接口提供使用，亦可以额外编写接口以满足开发需求。

## 标识码

标识码分为固定标识码和错误标识码，固定标识码为开发提供统一明了的。错误标识码需要返回前端并判断错误类型，提示用户错误发生的详细问题所在。