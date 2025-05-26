package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"example_client/internal/crypto_utils"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const mb100 = 104857600 

// использовать для нагруженных тестов. Здесь один чанк=100мб, не считая nonce + tag 
func StreamingUploadEncryptedFile(filePath, cloudURL, accessToken, category string, kEnc, kMac []byte) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Настраиваем AES-CBC и HMAC
	block, _ := aes.NewCipher(kEnc)
	iv := make([]byte, aes.BlockSize)
	rand.Read(iv)
	cbc := cipher.NewCBCEncrypter(block, iv)
	mac := hmac.New(sha256.New, kMac)
	mac.Write(iv) // первые байты HMAC

	// Создаём pipe
	pr, pw := io.Pipe()

	// Горутинa: читает файл -> шифрует -> pw.Write
	go func() {
		defer pw.Close()
		// Записываем nonce и iv
		nonce := make([]byte, 16)
		rand.Read(nonce)
		pw.Write(nonce)
		pw.Write(iv)

		// Шифруем блоками
		buf := make([]byte, mb100)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				// PKCS7 padding если конец файла
				if err == io.EOF {
					chunk = crypto_utils.Pkcs7Pad(chunk, aes.BlockSize)
				}
				ctext := make([]byte, len(chunk))
				cbc.CryptBlocks(ctext, chunk)

				mac.Write(ctext) // обновляем HMAC
				pw.Write(ctext)  // пишем шифртекст
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				pw.CloseWithError(err)
				return
			}
		}
		// записываем HMAC tag
		pw.Write(mac.Sum(nil))
	}()

	// Формируем запрос с chunked-Transfer-Encoding
	req, _ := http.NewRequest("POST", cloudURL, pr)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-File-Category", category)
	req.Header.Set("X-Orig-Filename", filepath.Base(filePath))
	req.Header.Set("X-Orig-Mime", "audio/x-psf")
	req.Header.Set("Content-Type", "application/octet-stream")
	// Content-Length мы не знаем заранее — пусть будет chunked

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("upload failed %d: %s", resp.StatusCode, body)
	}
	return nil
}

// uploadEncryptedFile — отправка зашифрованного blob-а. Использовать для тестирования исключительно небольших файлов
// так как из-за ioutil.ReadAll, файл сначала полностью загружается в ОЗУ.
func NotStreamingUploadEncryptedFile(filePath, cloudURL, accessToken, category string, kEnc, kMac []byte) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	blob, err := crypto_utils.BuildEncryptedBlob(content, kEnc, kMac)
	if err != nil {
		return nil, fmt.Errorf("encrypt blob: %w", err)
	}

	req, err := http.NewRequest("POST", cloudURL, bytes.NewReader(blob))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Orig-Filename", filepath.Base(filePath))
	req.Header.Set("X-Orig-Mime", mime.TypeByExtension(filepath.Ext(filePath)))
	req.Header.Set("X-File-Category", category)
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return body, fmt.Errorf("cloud API returned %d", res.StatusCode)
	}
	return body, nil
}
