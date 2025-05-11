gen-keys:
	@bash scripts/generate_keys.sh

client-run:
	go run client/main.go \
	-rsa-pub=keys/client_rsa.pub \
	-ecdsa-pub=keys/client_ecdsa.pub \
	-ecdsa-priv=keys/client_ecdsa.pem \
	-url=http://localhost:8080/handshake/init