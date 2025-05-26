package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// DownloadAndDecryptStream скачивает «encrypted_blob» по presignedURL,
// проверяет HMAC, расшифровывает AES-CBC блок за блоком, снимает PKCS#7-padding
func DownLoadFileByURL(presignedURL, outPath string, kEnc, kMac []byte) error {
	// создаём папку
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	resp, err := http.Get(presignedURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}
	// обязательно есть Content-Length
	cl := resp.Header.Get("Content-Length")
	total, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return err
	}
	// минимальный: 16+16+16+32 = 80 байт (nonce+iv+1 блок+tag)
	if total < int64(16+aes.BlockSize+aes.BlockSize+sha256.Size) {
		return fmt.Errorf("blob too small: %d", total)
	}

	// 1) nonce
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(resp.Body, nonce); err != nil {
		return err
	}

	// 2) iv
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(resp.Body, iv); err != nil {
		return err
	}
	// 3) ciphertext length
	cipherLen := total - int64(16+aes.BlockSize+sha256.Size)
	if cipherLen%aes.BlockSize != 0 {
		return fmt.Errorf("cipherLen not multiple of block: %d", cipherLen)
	}
	blocks := int(cipherLen / aes.BlockSize)

	// HMAC на iv||ciphertext
	macHasher := hmac.New(sha256.New, kMac)
	macHasher.Write(iv)

	blockCipher, err := aes.NewCipher(kEnc)
	if err != nil {
		return err
	}
	cbc := cipher.NewCBCDecrypter(blockCipher, iv)

	// открываем файл
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// читаем первый блок ciphertext
	buf := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(resp.Body, buf); err != nil {
		return err
	}
	macHasher.Write(buf)
	prev := make([]byte, aes.BlockSize)
	copy(prev, buf)

	// цикл по остальным блокам
	for i := 1; i < blocks; i++ {
		if _, err := io.ReadFull(resp.Body, buf); err != nil {
			return err
		}
		macHasher.Write(buf)
		// расшифровка prev
		tmp := make([]byte, aes.BlockSize)
		cbc.CryptBlocks(tmp, prev)
		if _, err := out.Write(tmp); err != nil {
			return err
		}
		copy(prev, buf)
	}
	// обрабатываем последний блок prev
	lastPlain := make([]byte, aes.BlockSize)
	cbc.CryptBlocks(lastPlain, prev)
	pad := int(lastPlain[aes.BlockSize-1])
	if pad < 1 || pad > aes.BlockSize {
		return fmt.Errorf("invalid padding: %d", pad)
	}
	// проверяем паддинг
	for i := 0; i < pad; i++ {
		if int(lastPlain[aes.BlockSize-1-i]) != pad {
			return fmt.Errorf("corrupt padding")
		}
	}
	// пишем все, кроме паддинга
	if _, err := out.Write(lastPlain[:aes.BlockSize-pad]); err != nil {
		return err
	}

	// 4) tag
	tag := make([]byte, sha256.Size)
	if _, err := io.ReadFull(resp.Body, tag); err != nil {
		return err
	}
	if !hmac.Equal(macHasher.Sum(nil), tag) {
		return fmt.Errorf("HMAC mismatch")
	}
	return nil
}
