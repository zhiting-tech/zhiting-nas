package filebrowser

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// 注意，涉及文件操作，并发测试可能会出错

var (
	testDataDir = "./data_test"
)

func setUp() {
	// setting.LoadConfig("../../app.yaml.test")
}

func tearDown() {
	// 涉及文件操作，手动删除会安全一点
	//if err := os.RemoveAll(GetFB().GetRoot()); err != nil {
	//	log.Fatal("clean up error, please delete test_data folder")
	//}
}

func TestCreateRemove(t *testing.T) {
	setUp()
	fb = GetFB()
	const fn = "file.md"

	_ = fb.Remove("/file.md")
	f, err := fb.Open(fn)
	assert.True(t, os.IsNotExist(err))

	f, err = fb.Create(fn)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	_, err = f.Write([]byte("hello world"))
	assert.NoError(t, err)
	err = f.Close()
	assert.NoError(t, err)
	assert.True(t, assert.FileExists(t, filepath.Join(testDataDir, fn)))

	fi, err := fb.Stat(fn)
	assert.NoError(t, err)
	assert.Equal(t, fi.Name(), "file.md")
	assert.Equal(t, fi.IsDir(), false)

	err = fb.Remove("/file.md")
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(testDataDir, fn))
	assert.True(t, os.IsNotExist(err))
	tearDown()
}

func TestMkdir(t *testing.T) {
	setUp()
	fb = GetFB()
	assert.NoError(t, fb.Mkdir("test"))
	assert.DirExists(t, filepath.Join(testDataDir, "test"))
	assert.NoError(t, fb.Mkdir("/test2"))
	assert.DirExists(t, filepath.Join(testDataDir, "test2"))
	assert.NoError(t, fb.Mkdir("test3/test4"))
	assert.DirExists(t, filepath.Join(testDataDir, "test3", "test4"))
	tearDown()
}
