gen-keys:
	@bash scripts/generate_keys.sh

run-client:
	go run ./cmd/main.go \
  -signup-url="http://localhost:8081/user/signup" \
  -email="12333@mail.ru" \
  -password="1234567" \
  -platform="tg-bot" \
  -rsa-pub="keys/client_rsa.pub" \
  -ecdsa-pub="keys/client_ecdsa.pub" \
  -ecdsa-priv="keys/client_ecdsa.pem" \
  -init-url="http://localhost:8080/handshake/init" \
  -fin-url="http://localhost:8080/handshake/finalize" \
  -session-test-url="http://localhost:8080/session/test" \
  -upload-file="test_photo/simple.jpg" \
  -cloud-url="http://localhost:8080/files/one/encrypted" \
  -category="photo"