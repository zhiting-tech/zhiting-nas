package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"io/ioutil"
	"os"
	"path"
)

// DecryptFile 使用给定的密钥解密由文件名指定的文件
func DecryptFile(pwd string, filename string, outFilename string) (string, error) {
	pwd = EncodeMD5(pwd)
	key := []byte(pwd)

	if len(outFilename) == 0 {
		outFilename = filename + ".dec"
	}

	ciphertext, err := ioutil.ReadFile(path.Join(config.AppSetting.UploadSavePath, filename))
	if err != nil {
		return "", err
	}

	of, err := os.Create(path.Join(config.AppSetting.UploadSavePath, outFilename))
	if err != nil {
		return "", err
	}
	defer of.Close()

	buf := bytes.NewReader(ciphertext)

	// ciphertext 在前 16 个字节中是IV，剩余部分为实际的密文
	iv := make([]byte, aes.BlockSize)
	if _, err = buf.Read(iv); err != nil {
		return "", err
	}

	// 密文的长度为加密文件内容长度减去Iv长度
	// 密文肯定是16字节的倍数，因为加密的时候有做过填充
	paddedSize := len(ciphertext) - aes.BlockSize
	if paddedSize%aes.BlockSize != 0 {
		return "", fmt.Errorf("密文错误")
	}

	plaintext := make([]byte, paddedSize)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext[aes.BlockSize:])

	// 减去填充的数量，获取到真正的明文长度
	bufLen := len(plaintext) - int(plaintext[len(plaintext)-1])

	if _, err := of.Write(plaintext[:bufLen]); err != nil {
		return "", err
	}
	// 输出文件
	return outFilename, nil
}

// DecryptString 解密字符串
func DecryptString(pwd, str string) (string, error) {
	// 把密码转成32位的字符串
	pwd = EncodeMD5(pwd)
	key := []byte(pwd)
	ciphertext, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", err
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
