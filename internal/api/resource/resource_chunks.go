package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"os"
	"strconv"
)

type ChunksInfosResp struct {
	Chunks []Chunk `json:"chunks"`
}

func GetChunks(c *gin.Context) {

	var (
		err  error
		resp ChunksInfosResp
	)

	defer func() {
		if len(resp.Chunks) == 0 {
			resp.Chunks = make([]Chunk, 0)
		}
		response.HandleResponse(c, err, resp)
	}()

	hash := c.Param("hash")

	user := session.Get(c)

	hashPath := fmt.Sprintf("/cache/%d/%s", user.UserID, hash)

	resp.Chunks, err = GetChunksInfos(hashPath)
}

func GetChunksInfos(hashPath string) (chunks []Chunk, err error) {
	fs := filebrowser.GetFB()

	file, err := fs.Open(hashPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return
		}
		err = errors.New(status.HashNotExistErr)
		return
	}

	fileInfos, err := file.Readdir(-1)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	for _, fileInfo := range fileInfos {
		chunksInfo := Chunk{
			Size: fileInfo.Size(),
		}

		id, _ := strconv.Atoi(fileInfo.Name())
		chunksInfo.ID = id

		chunks = append(chunks, chunksInfo)
	}

	return
}
