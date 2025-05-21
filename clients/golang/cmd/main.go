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

	// ecdsaPubDER, _ = base64.StdEncoding.DecodeString("MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEPwTK236EM2+nbKB17ovnR4W39uUZqw92iZ07DHYEabgQH9++zCOhrFM7uBqsPLbaTQiFvshctgloL5EzuiGVXw==")
	// //fmt.Println("\n\n\n",ecdsaPubDER, "\n\n\n")

	// privDER, _ := base64.StdEncoding.DecodeString("MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgi4onhfDcFPzHisg68efQ4z41LrGnz0PjY0mJASpmizahRANCAAQ/BMrbfoQzb6dsoHXui+dHhbf25RmrD3aJnTsMdgRpuBAf377MI6GsUzu4Gqw8ttpNCIW+yFy2CWgvkTO6IZVf")
	// ecdsaPriv, err = x509.ParseECPrivateKey(privDER)
	// if err != nil {
	// 	panic(err)
	// }
	// //fmt.Println("\n\n\n",ecdsaPriv, "\n\n\n")

	// rsaPubDER, _ = base64.StdEncoding.DecodeString("MIIBojANBgkqhkiG9w0BAQEFAAOCAY8AMIIBigKCAYEAvDqpMOyYkR7MAeEW4+xo1r6D6CmYbGdCAKuTo/NVQEvSZtaTEUham3seaBuA/1x9tLz8i12YpK9BolWG4HCFKG47o5iG7LuIMQ8kiSo7j+pV46tc9wUcilNj092rNMrL/xie6kKMaj/VCZsz2viamKwjMZHUlyAA2c/4EhWYn639YwVQOFaHoU43eBXgwLxQIYcx1OxBV+yMgsMFH9qLJKXuA9EmU0fHDZmjSUwUMwjVMs/kwSBsvJNND/R5ybPkLZX1kV0WTLX8Fhgeb0n1L/DIvZodNVbD2ymo8inZ3DEa2ooo5c/vs02fLKdDsvsuSsdDmG+qYIJKzZTLFmtLgYGY5nKZg8VPPW2IgktiybmfjqmOX5WwOaM7p/eL9nQgI36dRJRY9sw78Cz7gv3Uorxakw9HFM0+Nl3yY057MMflH+rd/uzW/OYGEaIOKlKT8vA2cAdSpkPaxs3bNB5+HLQFft8ChD0Wl+MKJtsQcFqBqJz/VkaEzhJtVnj4+3lZAgMBAAE=")
	// //fmt.Println("\n\n\n",rsaPubDER, "\n\n\n")
	// rsaPubDER123, _ := base64.StdEncoding.DecodeString("MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgtfqBe45kU6WU6CLtvBHtn5Be2+fyihMu5lPydjmGRmyhRANCAARkcKjY4No6R2kLxeVC58byA+h4LnPdkksHKDWjJ58H7G+uPFMSpBGWk+YaKwId8Y9ajdz2ezlFb6IASefXfeJS")
	// fmt.Println("\n\n\n",rsaPubDER123, "\n\n\n")

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
