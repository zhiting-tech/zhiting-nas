package folder

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"google.golang.org/grpc"
	"io/ioutil"
	"os/exec"
	"path"
	"time"
)

type removeAreaReq struct {
	IsDelCloudDisk *bool `json:"is_del_cloud_disk"`
}

type removeAreaResp struct {
	RemoveStatus int `json:"remove_status"`
}

var removingStatusChan = make(chan int, 1)
var isRemovingChan = make(chan struct{}, 1)

func RemoveArea(c *gin.Context) {
	var (
		err  error
		req  removeAreaReq
		resp removeAreaResp
	)
	defer func() {
		response.HandleResponse(c, err, resp)
	}()

	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	user := session.Get(c)
	// check Is it the owner
	if err = req.validateRequest(user.IsOwner); err != nil {
		return
	}

	if len(removingStatusChan) == 0 && len(isRemovingChan) == 0 {
		go req.remove(user.AreaName)
	}
	// timeout
	timeout := time.After(time.Second * 1)
	// 3 success| 1 removing| 2 error
	select {
	case <-timeout:
		resp.RemoveStatus = 1
	case s := <-removingStatusChan:
		resp.RemoveStatus = s
		if resp.RemoveStatus == 2 {
			err = errors.Wrap(err, errors.InternalServerErr)
		}
	}
}

func removeArea(isRemoveFile bool, areaName string) error {
	var (
		fs  = filebrowser.GetFB()
		err error
	)
	areaName = fmt.Sprintf("%s_%d", areaName, time.Now().Unix())
	// 查找出所有存储池
	poolNameMap, err := getAllPoolName()
	if err != nil {
		return err
	}
	dir, err := ioutil.ReadDir(config.AppSetting.UploadSavePath)
	if err != nil {
		return err
	}
	// 1 remove| 0 notRemove
	if isRemoveFile == true {
		// 移除该家庭的所有数据库记录
		entity.DelAllFolderRecode()
		entity.DelAllFolderAuthRecode()
		// 移除所有文件
		for _, v := range dir {
			if _, ok := poolNameMap[v.Name()]; ok {
				if err = delPool(v.Name()); err != nil {
					return err
				}
				continue
			}
			if err = forceDel(path.Join(config.AppSetting.UploadSavePath, v.Name())); err != nil {
				config.Logger.Info("remove err:", err)
				return err
			}
		}
	} else if isRemoveFile == false {
		// TODO 使用上个家庭的数据时可能需要用到就数据库数据 这里先统一删除
		// 创建以家庭名称命名的文件 将所有volum的文件复制过去，然后删除除家庭xxx外所有文件，清除所有存储池
		entity.DelAllFolderRecode()
		entity.DelAllFolderAuthRecode()
		// 修改所有文件文件名称 格式为.old_areName_xx
		oldAreaFilePath := fmt.Sprintf(".%s_%d", areaName, time.Now().Unix())
		if err = fs.Mkdir(oldAreaFilePath); err != nil {
			return err
		}
		for _, v := range dir {
			if v.Name() == oldAreaFilePath {
				continue
			}
			if err = fs.CopyDir(path.Join(config.AppSetting.UploadSavePath, v.Name()), path.Join(config.AppSetting.UploadSavePath, oldAreaFilePath)); err != nil {
				return err
			}
			if _, ok := poolNameMap[v.Name()]; ok {
				if err = delPool(v.Name()); err != nil {
					return err
				}
				continue
			}
			if err = forceDel(path.Join(config.AppSetting.UploadSavePath, v.Name())); err != nil {
				config.Logger.Info("remove err:", err)
				return err
			}
		}
	}
	return err
}

func (req *removeAreaReq) remove(areaName string) {
	defer func() {
		<-isRemovingChan
	}()
	isRemovingChan <- struct{}{}
	if err := removeArea(*req.IsDelCloudDisk, areaName); err != nil {
		removingStatusChan <- 2
	} else {
		removingStatusChan <- 3
	}
}

func (req *removeAreaReq) validateRequest(isOwner bool) (err error) {
	// 判断是否有拥有者权限
	if isOwner != true {
		return errors.Wrap(err, status.ResourceNotCopyErr)
	}
	if req.IsDelCloudDisk == nil {
		isDelCloudDisk := false
		req.IsDelCloudDisk = &isDelCloudDisk
	}
	return
}

func forceDel(path string) error {
	var out bytes.Buffer
	var stderr bytes.Buffer
	var (
		err error
	)
	_, err = exec.LookPath("rm")
	if err != nil {
		return err
	}
	cmd := exec.Command("rm", "-rf", path)
	if err = cmd.Run(); err != nil {
		fmt.Println("forceDel Run err:", fmt.Sprint(err)+": "+stderr.String())
		return err
	}
	fmt.Println("Result: " + out.String())
	return err

}

func delPool(name string) error {
	// 创建通道
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}

	defer conn.Close()

	// 创建连接
	client := proto.NewDiskManagerClient(conn)

	param := proto.VolumeGroupRemoveReq{
		VGName: name,
	}
	ctx := context.Background()
	// 调用服务端方法
	result, err := client.VolumeGroupRemove(ctx, &param)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Error("VolumeGroupRemove error: %v", err)
		return err
	}
	config.Logger.Infof("VolumeGroupRemove info: %v", result)
	return nil
}

func getAllPoolName() (map[string]struct{}, error) {
	var (
		tmpMap = make(map[string]struct{})
	)

	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	groups, err := client.VolumeGroupList(ctx, &proto.Empty{})
	if err != nil {
		return nil, err
	}
	for _, v := range groups.VGS {
		if v.Name == types.LvmSystemDefaultName {
			continue
		}
		tmpMap[v.Name] = struct{}{}

	}
	return tmpMap, nil

}
