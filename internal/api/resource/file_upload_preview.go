package resource

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"log"
	"net/http"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
)

type PreviewResp struct{}

func runFuncName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}
func FilePreview(c *gin.Context) {
	var (
		resp PreviewResp
		err  error
	)
	fb := filebrowser.GetFB()

	id, err := strconv.Atoi(c.Param("id"))
	log.Print(fmt.Sprintf("%s_%s", runFuncName, "AtoiId"), id)
	if err != nil {
		log.Print(fmt.Sprintf("%s_%s", runFuncName, "Atoi err:"), err)
		err = errors.Wrap(err, errors.BadRequest)
		response.HandleResponse(c, err, &resp)
		return
	}

	folderInfo, err := entity.GetFolderInfo(id)
	log.Print(fmt.Sprintf("%s_%s", runFuncName, "GetFolderInfo"), folderInfo)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		response.HandleResponse(c, err, &resp)
		return
	}

	ext := utils.GetPathExt(folderInfo.AbsPath)
	ext = strings.ToLower(ext)
	log.Print(fmt.Sprintf("%s_%s", runFuncName, "ext :"), ext)
	fileType, ok := FileTypeMap[ext]
	if !ok || (fileType != types.FolderOfficeWordPPt && fileType != types.FolderOfficeExcel) {
		return
	}

	filePath, exists := PathExists(utils.FilePath(folderInfo.Hash)+"/"+strings.Split(folderInfo.Name, ".")[0], fileType)
	if exists == true {
		log.Println(fmt.Sprintf("%s_%s", runFuncName, "open :"), filePath)

		open, err := fb.Open(filePath)
		if err != nil {
			log.Println(fmt.Sprintf("%s_%s", runFuncName, "true open err:"), err)
			return
		}
		defer open.Close()

		stat, err := fb.Stat(filePath)
		log.Println(fmt.Sprintf("%s_%s", runFuncName, "stat :"), stat)
		if err != nil {
			return
		}

		http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), open)
		return
	} else {
		log.Println(fmt.Sprintf("%s_%s", runFuncName, "convert src-des :"), path.Join(fb.GetRoot(), folderInfo.AbsPath), path.Join(fb.GetRoot(), utils.FilePath(folderInfo.Hash)))
		convert := Convert(path.Join(fb.GetRoot(), folderInfo.AbsPath), path.Join(fb.GetRoot(), utils.FilePath(folderInfo.Hash)), fileType)
		log.Println(fmt.Sprintf("%s_%s", runFuncName, "convert :"), convert)
		if convert != "" {
			open, err := fb.Open(filePath)
			if err != nil {
				log.Println(fmt.Sprintf("%s_%s", runFuncName, "open err:"), err)
				return
			}
			defer open.Close()

			stat, err := fb.Stat(filePath)
			if err != nil {
				return
			}

			http.ServeContent(c.Writer, c.Request, stat.Name(), stat.ModTime(), open)
			return
		}
	}
}

func PathExists(path string, fileType int) (string, bool) {
	fb := filebrowser.GetFB()
	if fileType == types.FolderOfficeWordPPt {
		path = path + ".pdf"
	} else {
		path = path + ".html"
	}
	log.Println(fmt.Sprintf("%s_%s", runFuncName, "stat path :"), path)

	_, err := fb.Stat(path)
	if err == nil {
		return path, true
	}
	log.Println(fmt.Sprintf("%s_%s", runFuncName, "stat err :"), err)
	return path, false
}

func interactiveToExec(commandName string, params []string) (string, bool) {
	cmd := exec.Command(commandName, params...)
	buf, err := cmd.Output()
	w := bytes.NewBuffer(nil)
	cmd.Stderr = w
	if err != nil {
		log.Println("Error: <", err, "> when exec command read out buffer")
		return "", false
	} else {
		return string(buf), true
	}
}

func Convert(srcPath, desPath string, fileType int) string {
	fb := filebrowser.GetFB()
	fb.GetRoot()
	commandName := ""
	var params []string
	var fType string
	if fileType == types.FolderOfficeWordPPt {
		fType = "pdf"
	} else {
		fType = "html"
	}
	if runtime.GOOS == "windows" {
		commandName = "cmd"
		params = []string{"/c", "soffice", "--headless", "--invisible", "--convert-to", fType, "--outdir", desPath, srcPath}
	} else if runtime.GOOS == "linux" {
		commandName = "libreoffice"
		params = []string{"--invisible", "--headless", "--convert-to", fType, "--outdir", desPath, srcPath}
	}
	if _, ok := interactiveToExec(commandName, params); ok {
		resultPath := desPath + "/" + strings.Split(path.Base(srcPath), ".")[0]  + ".pdf"
		return resultPath
	} else {
		return ""
	}
}
