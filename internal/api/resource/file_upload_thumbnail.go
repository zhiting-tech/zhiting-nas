package resource

import (
	"bytes"
	"fmt"
	"github.com/disintegration/imaging"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/strukturag/libheif/go/heif"
)

var (
	FileTypeMap = make(map[string]int)
)

// generationThumbnail 图片或视频缩略图生成
func generationThumbnail(flag bool, path, hash string, ty int) error {
	// 加密带后缀.enc
	fb := filebrowser.GetFB()
	if flag == true {
		path = path + types.FolderEncryptExt
	}

	hash = strings.Split(hash, "-")[0]
	var err error
	// 本地存储
	if err = fb.Mkdir(utils.FileDir(hash) /*, os.ModePerm*/); err != nil {
		log.Print("generationThumbnail os.MkdirAll failed:", err)
		return err
	}

	des, err := fb.Create(utils.FilePath(hash) + ".png")
	if err != nil {
		log.Print("generationThumbnail os.Create failed:", err)
		return err
	}
	defer des.Close()

	if ty == types.FolderPhoto {
		// 图片格式
		err = photoThumbnail(path, filepath.Join(config.AppSetting.UploadSavePath, utils.FilePath(hash)+".png"))
		return err
	} else if ty == types.FolderVideo {
		// 视频格式
		err = videoThumbnail(path, filepath.Join(config.AppSetting.UploadSavePath, utils.FilePath(hash)+".png"))
		return err
	} else {
		// 什么都不是
		log.Print("generationThumbnail type error")
		return fmt.Errorf("error type!")
	}
}

// photoThumbnail photo thumbnail generation
func photoThumbnail(fileName, path string) error {
	var err error
	open, err := imaging.Open(fileName)
	if err != nil {
		log.Print("photoThumbnail imaging.Open failed:", err)
		return err
	}
	thumbnail := imaging.Thumbnail(open, 100, 100, imaging.Lanczos)

	if err = imaging.Save(thumbnail, path); err != nil {
		log.Print("photoThumbnail imaging.Save failed:", err)
		return err
	}

	return nil
}

// videoThumbnail video thumbnail generation
func videoThumbnail(fileName, path string) error {
	var err error
	if err = ffmpegScreenshot(fileName, path); err != nil {
		return err
	}
	log.Println("videoThumbnail path:", path)
	tmpPath := path
	if err = photoThumbnail(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func ffmpegScreenshot(fileName, dstName string) error {
	tmp := []string{"-ss", "1", "-t", "0.01", "-i", fileName, "-vf", "scale=160:-2", "-y", dstName}
	cmd := exec.Command("ffmpeg", tmp...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	log.Println("Result: " + out.String())
	return nil
}
