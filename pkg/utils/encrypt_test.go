package utils_test

import (
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"testing"
)

func TestEncryptString(t *testing.T) {
	encryptString, err := utils.EncryptString("123456", "123456")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(encryptString)
}


