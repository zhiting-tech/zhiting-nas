package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"io"
	"io/ioutil"
	"os"
	"path"
)

// EncryptFile 对文件进行加密， 加密方式：aes-cbc，填充算法为PKCS#7
func EncryptFile(pwd string, filename string, outFilename string) (string, error) {
	pwd = EncodeMD5(pwd)
	key := []byte(pwd)

	if len(outFilename) == 0 {
		outFilename = filename + ".enc"
	}

	plaintext, err := ioutil.ReadFile(path.Join(config.AppSetting.UploadSavePath, filename))
	if err != nil {
		return "", err
	}

	of, err := os.Create(path.Join(config.AppSetting.UploadSavePath, outFilename))
	if err != nil {
		return "", err
	}
	defer of.Close()

	// 如果原文长度不是16字节的倍数(如果刚好是16 的倍数，也需要填充16个)
	// 使用PKCS#7填充方式去填充
	// 缺几个字节就填几个缺的字节数
	bytesToPad := aes.BlockSize
	if len(plaintext)%aes.BlockSize != 0 {
		//  需要填充的数目
		bytesToPad = aes.BlockSize - (len(plaintext) % aes.BlockSize)
	}
	// 生成填充字节数组，每个填充的字节内容为缺的字节数
	padding := bytes.Repeat([]byte{byte(bytesToPad)}, bytesToPad)
	plaintext = append(plaintext, padding...)

	// 生成IV向量写入到输出的文件中，固定是开头的16字节长度
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	if _, err = of.Write(iv); err != nil {
		return "", err
	}

	// 密文与填充后的明文大小相同
	ciphertext := make([]byte, len(plaintext))

	//  使用 cipher.Block 接口的 AES 实现来加密整个文件
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	if _, err = of.Write(ciphertext); err != nil {
		return "", err
	}
	return outFilename, nil
}

// EncryptString 加密字符串
func EncryptString(pwd, str string) (string, error) {
	// 把密码转成32位的字符串
	pwd = EncodeMD5(pwd)
	key := []byte(pwd)

	text := []byte(str)
	c, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	return hex.EncodeToString(gcm.Seal(nonce, nonce, text, nil)), nil
}
