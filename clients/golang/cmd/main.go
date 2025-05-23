package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"example_client/internal/client"
	"example_client/internal/dto"
	fileloader "example_client/internal/file_loader"
	"example_client/internal/utils"
)

func main() {
	// === Флаги ===
	// для регистрации
	signupURL := flag.String("signup-url", "http://localhost:8081/user/signup", "")
	email := flag.String("email", "", "Email для регистрации")
	password := flag.String("password", "", "Password для регистрации")
	platform := flag.String("platform", "tg-bot", "Platform (web или tg-bot)")

	// для handshake и session-test
	rsaPubPath := flag.String("rsa-pub", "keys/client_rsa.pub", "")
	ecdsaPubPath := flag.String("ecdsa-pub", "keys/client_ecdsa.pub", "")
	ecdsaPrivPath := flag.String("ecdsa-priv", "keys/client_ecdsa.pem", "")
	initURL := flag.String("init-url", "http://localhost:8080/handshake/init", "")
	finURL := flag.String("fin-url", "http://localhost:8080/handshake/finalize", "")
	sesURL := flag.String("session-test-url", "http://localhost:8080/session/test", "")

	// для загрузки файла
	uploadFile := flag.String("upload-file", "", "Путь до локального файла")
	cloudURL := flag.String("cloud-url", "http://localhost:8080/files/one/encrypted", "URL эндпоинта загрузки")
	category := flag.String("category", "photo", "Категория файла: photo|video|text|unknown")

	flag.Parse()

	// === 1. SignUp: регистрация и получение access_token ===
	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Обязательно передать флаги -email и -password для регистрации")
		os.Exit(1)
	}
	accessToken, refreshToken, err := client.DoSignUpAPI(*signupURL, *email, *password, *platform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SignUp failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Успешная регистрация. Получен access_token:", accessToken)
	fmt.Println("Получен refresh_token (для platform=tg-bot):", refreshToken)

	// === 2. Загружаем ключи клиента ===
	rsaPubDER, err := fileloader.LoadDERPub(*rsaPubPath)
	if err != nil {
		panic(err)
	}
	ecdsaPubDER, err := fileloader.LoadDERPub(*ecdsaPubPath)
	if err != nil {
		panic(err)
	}
	ecdsaPriv, err := fileloader.LoadECDSAPriv(*ecdsaPrivPath)
	if err != nil {
		panic(err)
	}

	// === 3. Init Handshake (с заголовком Authorization) ===
	startInit := time.Now()
	initResp, err := client.DoInitAPI(*initURL, rsaPubDER, ecdsaPubDER, ecdsaPriv, accessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init Handshake failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Init resp: %+v\n", initResp)
	fmt.Println("Init handshake time:", time.Since(startInit))

	// === 4. Finalize Handshake (с заголовком Authorization) ===
	startFin := time.Now()
	session, err := client.DoFinalizeAPI(*finURL, *sesURL, initResp, ecdsaPriv, accessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Finalize Handshake failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Finalize handshake time:", time.Since(startFin))

	// === 5. Session Test (с заголовком Authorization) ===
	startSesTest := time.Now()
	if err := session.DoSessionTest(generateBigMsg(1024)); err != nil {
		panic(err)
	}
	fmt.Println("Session test time:", time.Since(startSesTest))

	// === 6. Если указан -upload-file, шифруем и отправляем файл ===
	var rBody []byte
	if *uploadFile != "" {
		fmt.Printf("Загружаем файл «%s» в зашифрованном виде на %s …\n", *uploadFile, *cloudURL)
		respBody, err := uploadEncryptedFile(
			*uploadFile,
			*cloudURL,
			accessToken,
			*category,
			session.KEnc,
			session.KMac,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "uploadEncryptedFile error: %v\n", err)
			os.Exit(1)
		}
		rBody = respBody
		fmt.Println("Ответ от cloud-API:\n", string(respBody))
	}

	var fileResp dto.FileResponse
	if err := json.Unmarshal(rBody, &fileResp); err != nil {
		fmt.Fprintf(os.Stderr, "не удалось разобрать JSON ответа: %v\n", err)
		os.Exit(1)
	}

	outDir := "out_dir/downloaded_photo.jpg"

	if err := utils.DownloadAndDecryptStream(fileResp.Url, outDir, session.KEnc, session.KMac); err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		os.Exit(1)
	}
}

// generateBigMsg генерирует base64-строку размером sizeBytes.
// В примере используется для session-test.
func generateBigMsg(sizeBytes int) string {
	b := make([]byte, sizeBytes)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// uploadEncryptedFile строит пакет данныx и отправляет его «сырыми байтами» в облако.
func uploadEncryptedFile(
	filePath, cloudURL, accessToken, category string,
	kEnc, kMac []byte,
) ([]byte, error) {
	// Открываем файл
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer f.Close()

	// Читаем содержимое
	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	// Определяем имя и mime-тип
	origName := filepath.Base(filePath)
	ext := filepath.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = http.DetectContentType(fileBytes)
	}

	// Строим «зашифрованный blob»
	encryptedBlob, err := buildEncryptedBlob(fileBytes, kEnc, kMac)
	if err != nil {
		return nil, fmt.Errorf("ошибка при шифровании: %w", err)
	}

	// Делаем POST-запрос
	req, err := http.NewRequest("POST", cloudURL, bytes.NewReader(encryptedBlob))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Orig-Filename", origName)
	req.Header.Set("X-Orig-Mime", mimeType)
	req.Header.Set("X-File-Category", category)
	req.Header.Set("Content-Type", "application/octet-stream")

	clientHTTP := &http.Client{Timeout: 10 * time.Minute}
	resp, err := clientHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("cloud-API вернул статус %d", resp.StatusCode)
	}
	return body, nil

}

// buildEncryptedBlob собирает пакет:
//
//	timestamp(8) || nonce(16) || IV(16) || ciphertext || tag(32)
//
// где ciphertext = AES-256-CBC(PKCS#7(plain)), а tag = HMAC-SHA256(iv || ciphertext).
func buildEncryptedBlob(
	plain, kEnc, kMac []byte,
) ([]byte, error) {
	// 1) timestamp (8 байт)
	now := time.Now().UnixMilli()
	tsBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBuf, uint64(now))

	// 2) nonce (16 байт)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// 3) IV (16 байт)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	// 4) PKCS#7 + AES-CBC
	block, err := aes.NewCipher(kEnc)
	if err != nil {
		return nil, err
	}
	padded := client.Pkcs7Pad(plain, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	// 5) tag = HMAC-SHA256(iv || ciphertext)
	mac := hmac.New(sha256.New, kMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	tag := mac.Sum(nil) // 32 байта

	// 6) Собираем весь буфер
	buf := bytes.Buffer{}
	buf.Write(tsBuf)
	buf.Write(nonce)
	buf.Write(iv)
	buf.Write(ciphertext)
	buf.Write(tag)

	return buf.Bytes(), nil
}
