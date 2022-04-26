package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/folder"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"google.golang.org/grpc"
	"strconv"
	"time"
)

func GetUserInfo(token string) (result *GetUserInfoResp, err error) {
	conn, err := grpc.Dial(config.ExtServerSetting.SaServer, grpc.WithInsecure())
	if err != nil {
		return
	}
	defer conn.Close()

	client := NewExtensionClient(conn)
	ctx := context.Background()
	req := &GetAreaInfoReq{Token: token}
	result, err = client.GetUserInfo(ctx, req)
	if err != nil {
		return
	}
	return
}

func SANotifyEvent() {
loop:
	conn, err := grpc.Dial(config.ExtServerSetting.SaServer, grpc.WithInsecure())
	if err != nil {
		goto loop
	}
	defer conn.Close()
	client := NewExtensionClient(conn)
	ctx := context.Background()
	empty := &EmptyReq{}
	stream, err := client.SANotifyEvent(ctx, empty)
	if err != nil {
		//return err
	}
	for {
		reConnect := func() (*SAEventInfo, error) {
			var resp = &SAEventInfo{}
			var er error
			defer func(er error) {
				if err := recover(); err != nil {
					er = fmt.Errorf("stream.Recv() panic")
				}
			}(er)
			resp, er = stream.Recv()
			return resp, er
		}
		resp, err := reConnect()
		if resp == nil || err != nil {
			break
		}
		go executive(resp)
	}

	fmt.Printf("after 10 sec rpc to connect:%s again", config.ExtServerSetting.SaServer)
	time.Sleep(time.Second * 10)
	goto loop
}

type saEventId struct {
	Ids []int `json:"ids"`
}

func executive(saEvent *SAEventInfo) (err error) {
	var ids saEventId
	if err = json.Unmarshal(saEvent.Data, &ids); err != nil {
		return
	}
	if saEvent.Event == types.SaDepartmentRemoveEvent {
		for _, v := range ids.Ids {
			id := entity.QueryFolderByUid(-v)
			req := folder.DelReq{Id: id.ID}
			task.GetTaskManager().Add(types.TaskDelFolder, strconv.Itoa(req.Id), &req)
		}
	} else if saEvent.Event == types.SaUserRemoveEvent {
		if err = delUser(ids.Ids); err != nil {
			return
		}
	}

	return
}

func delUser(uIds []int) error {
	fs := filebrowser.GetFB()
	// 用户退出/被移除家庭或企业是否自动删除私人文件夹和其它文件
	if config.AppSetting.IsAutoDel == 0 {
		return nil
	}
	// 查找私人文件
	folderInfos, err := entity.GetPrivateFolders(uIds)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return err
	}
	// 移除私人文件
	for _, folderInfo := range folderInfos {
		err = utils.RemoveFolderAndRecode(fs, folderInfo.AbsPath)
		if err != nil {
			return err
		}
	}

	// 查找和移除用户初始化生成的个人文件
	for _, v := range uIds {
		folderRow, err := entity.GetRelateFolderInfoByUid(types.FolderSelfDirUid, v)
		if err != nil {
			return err
		}
		err = utils.RemoveFolderAndRecode(fs, folderRow.AbsPath)
		if err != nil {
			return err
		}
	}

	// 删除属于用户uid的所有权限Auth
	if err = entity.DelFolderAuthByUid(uIds); err != nil {
		err = errors.Wrap(err, status.FolderRemoveErr)
		return err
	}
	return nil
}
