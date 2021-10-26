package utils_test

import (
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"testing"
)

func TestDecryptString(t *testing.T) {
	encryptString, err := utils.DecryptString("123456", "bdd7101c3884b04e9ce0fde05fbb210509817e797505e58f178fff4a481db7b4da43")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(encryptString)
}
