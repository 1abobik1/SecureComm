package main

import (
	"crypto/rand"
	"encoding/base64"
	"example_client/internal/client"
	fileloader "example_client/internal/file_loader"
	"flag"
	"fmt"
	"time"
)

func main() {
	// Flags
	rsaPubPath := flag.String("rsa-pub", "keys/client_rsa.pub", "")
	ecdsaPubPath := flag.String("ecdsa-pub", "keys/client_ecdsa.pub", "")
	ecdsaPrivPath := flag.String("ecdsa-priv", "keys/client_ecdsa.pem", "")
	initURL := flag.String("init-url", "http://localhost:8080/handshake/init", "")
	finURL := flag.String("fin-url", "http://localhost:8080/handshake/finalize", "")
	SesURL := flag.String("session-test-url", "http://localhost:8080/session/test", "")

	flag.Parse()

	// Load files
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

	// Init
	startInit := time.Now()
	initResp := client.DoInitAPI(*initURL, rsaPubDER, ecdsaPubDER, ecdsaPriv)
	fmt.Printf("Init resp: %+v\n", initResp)
	fmt.Println("\nInit handshake time:", time.Since(startInit))

	// Finalize
	startFin := time.Now()
	session := client.DoFinalizeAPI(*finURL, *SesURL, initResp, ecdsaPriv)
	fmt.Println("\nFinalize handshake time:", time.Since(startFin))

	// test сессии, путем отправки тестового сообщения
	startSesTest := time.Now()
	if err := session.DoSessionTest(generateBigMsg(mb10)); err != nil {
		panic(err)
	}
	fmt.Println("\nSession test time:", time.Since(startSesTest))

}

const mb10 = 10485760

func generateBigMsg(sizeBytes int) string {
	b := make([]byte, sizeBytes)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
