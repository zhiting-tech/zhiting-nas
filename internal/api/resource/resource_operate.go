package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"os"
	"path/filepath"
	"time"
)

const (
	copyAction = "copy"
	moveAction = "move"
)

type OperateResourceReq struct {
	Action         string   `json:"action"`
	Destination    string   `json:"destination"`
	Sources        []string `json:"sources"`
	DestinationPwd string   `json:"destination_pwd"`
	IsEncryptMove  []int	// 是否加密路径内移动
}

// FolderResource 信息
type FolderResource struct {
	realPath   string             // 实际路径
	folderId   int                // 根目录folderId
	folderInfo *entity.FolderInfo // 根目录文件信息
	folderAuth *entity.FolderAuth // 根目录文件权限信息
}

func OperateResource(c *gin.Context) {
	var (
		err error
		req OperateResourceReq
	)

	defer func() {
		response.HandleResponse(c, err, nil)
	}()

	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	user := session.Get(c)

	// 检查参数
	if err = req.validateRequest(user.UserID); err != nil {
		return
	}

	// 操作文件
	if err = req.OperateResource(user.UserID); err != nil {
		return
	}
}

// OperateResource 操作资源
func (req *OperateResourceReq) OperateResource(uid int) (err error) {
	fs := filebrowser.GetFB()
	secret, err := utils.GetFolderSecret(req.Destination, req.DestinationPwd)
	if err != nil {
		return err
	}

	req.Destination, err = utils.GetNewPath(req.Destination)
	if err != nil {
		return
	}

	for key, path := range req.Sources {
		switch req.Action {
		case moveAction:
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
		default:
			// 复制文件和目录
			if err = req.Copy(path, fs); err != nil {
				config.Logger.Errorf("resource_operate Copy err %v", err)
				return
			}
		}

		// 保存文件夹数据
		// 把key也带入，以便找出是否是加密路径内的移动
		destPath := filepath.Join(req.Destination, filepath.Base(path))
		if err = req.saveFolder(key, uid, secret, destPath); err != nil {
			config.Logger.Errorf("resource_operate save folder err %v", err)
			return
		}
	}
	return
}

// Copy 复制操作
func (req *OperateResourceReq) Copy(path string, fs *filebrowser.FileBrowser) (err error) {
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
	return
}

// encryptFolder 处理加密文件夹
func (req *OperateResourceReq) saveFolder(key, uid int, secret, source string) (err error) {
	var files []os.FileInfo
	var folders []*entity.FolderInfo

	fs := filebrowser.GetFB()
	isDir, err := fs.IsDir(source)
	if err != nil {
		return
	}

	// 判断是否为目录
	if isDir {
		srcFile, _ := fs.Open(source)
		// 记录目录信息
		folders = append(folders, &entity.FolderInfo{Name: filepath.Base(source), Type: types.FolderTypeDir, AbsPath: source})
		// 遍历当前目录
		files, err = srcFile.Readdir(-1)
		if err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
		// 循环目录下的所有文件
		for _, file := range files {
			fSource := source + "/" + file.Name()
			if file.IsDir() {
				// 如果是目录，记录目录信息
				folders = append(folders, &entity.FolderInfo{Name: file.Name(), Type: types.FolderTypeDir, AbsPath: fSource})
				// 递归循环目录下的数据
				err = req.saveFolder(key, uid, secret, fSource)
				if err != nil {
					return
				}
			} else {
				// 如果是文件，记录文件信息
				folders = append(folders, &entity.FolderInfo{Name: file.Name(), Type: types.FolderTypeFile, AbsPath: fSource})

				// 加密文件
				if err = req.encryptFile(key, secret, fSource, fs); err != nil {
					return
				}
			}
		}
	} else {
		// 如果是文件，记录文件信息
		folders = append(folders, &entity.FolderInfo{Name: filepath.Base(source), Type: types.FolderTypeFile, AbsPath: source})

		// 加密文件
		if err = req.encryptFile(key, secret, source, fs); err != nil {
			return
		}
	}

	// 如果是复制，需要插入folders
	if req.Action == copyAction {
		// 批量写入folder信息
		for i := range folders {
			folders[i].CreatedAt = time.Now().Unix()
			folders[i].Uid = uid
		}
		if err = entity.BatchInsertFolder(entity.GetDB(), folders); err != nil {
			return
		}
	}

	return
}

// encryptFile 加密文件
func (req *OperateResourceReq) encryptFile(key int, secret, source string, fs *filebrowser.FileBrowser) (err error) {
	// 没有密钥或者是加密路径下的移动，不处理
	if secret == "" || req.IsEncryptMove[key] == 1 {
		return
	}
	if _, err = utils2.EncryptFile(secret, source, source + types.FolderEncryptExt); err != nil {
		return
	}
	if err = fs.Remove(source); err != nil {
		return
	}
	if err = fs.Rename(source + types.FolderEncryptExt, source); err != nil {
		return
	}
	return
}