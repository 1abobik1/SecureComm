package main

import (
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

	startSesTest := time.Now()
	if err := session.DoSessionTest("Hello, secure session!"); err != nil {
		panic(err)
	}
	fmt.Println("\nSession test time:", time.Since(startSesTest))

}
