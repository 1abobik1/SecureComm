package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"example_client/internal/client"
	fileloader "example_client/internal/file_loader"
)

func main() {
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

	// регистрация и получение access_token
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

	// Загружаем ключи клиента
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

	// Init Handshake (с заголовком Authorization)
	startInit := time.Now()
	initResp, err := client.DoInitAPI(*initURL, rsaPubDER, ecdsaPubDER, ecdsaPriv, accessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Init Handshake failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Init resp: %+v\n", initResp)
	fmt.Println("Init handshake time:", time.Since(startInit))

	// Finalize Handshake (с заголовком Authorization)
	startFin := time.Now()
	session, err := client.DoFinalizeAPI(*finURL, *sesURL, initResp, ecdsaPriv, accessToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Finalize Handshake failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Finalize handshake time:", time.Since(startFin))

	// session test
	// startSesTest := time.Now()
	// if err := session.DoSessionTest(generateBigMsg(1024)); err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Session test time:", time.Since(startSesTest))

	// здесь можно использовать методы NotStreamingUploadEncryptedFile(для загрузки мелких файлов) или StreamingUploadEncryptedFile(для загрузки больших файлов)
	//var rBody []byte
	if *uploadFile != "" {
		fmt.Printf("Загружаем файл «%s» в зашифрованном виде на %s …\n", *uploadFile, *cloudURL)
		respBody, err := client.NotStreamingUploadEncryptedFile(*uploadFile, *cloudURL, accessToken, *category, session.KEnc, session.KMac)
		if err != nil {
			fmt.Fprintf(os.Stderr, "uploadEncryptedFile error: %v\n", err)
			os.Exit(1)
		}
		rBody = respBody
		fmt.Println("Ответ от cloud-API:\n", string(respBody))
	}

	// var fileResp dto.FileResponse
	// if err := json.Unmarshal(rBody, &fileResp); err != nil {
	// 	fmt.Fprintf(os.Stderr, "не удалось разобрать JSON ответа: %v\n", err)
	// 	os.Exit(1)
	// }

	// outDir := "out_dir/downloaded_photo.jpg"

	// if err := utils.DownLoadFileByURL(fileResp.Url, outDir, session.KEnc, session.KMac); err != nil {
	// 	fmt.Printf("Ошибка: %v\n", err)
	// 	os.Exit(1)
	// }
}

// generateBigMsg генерирует base64-строку размером sizeBytes.
// В примере используется для session-test.
func generateBigMsg(sizeBytes int) string {
	b := make([]byte, sizeBytes)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
