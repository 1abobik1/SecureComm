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
	// finURL := flag.String("fin-url", "http://localhost:8080/handshake/finalize", "")
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

	startInit := time.Now()
	// Step 1: Init
	initResp := client.DoInitAPI(*initURL, rsaPubDER, ecdsaPubDER, ecdsaPriv)
	fmt.Printf("Init resp: %+v\n", initResp)
	fmt.Println("\nInit handshake time:", time.Since(startInit))

	// Step 2: Finalize
	// client.DoFinalizeAPI(*finURL, initResp, ecdsaPriv)
}
