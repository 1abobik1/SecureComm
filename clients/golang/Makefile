gen-keys:
	@bash scripts/generate_keys.sh

run-client:
	go run cmd/main.go \
	-rsa-pub=keys/client_rsa.pub \
	-ecdsa-pub=keys/client_ecdsa.pub \
	-ecdsa-priv=keys/client_ecdsa.pem \
	-init-url=http://localhost:8080/handshake/init \
	-fin-url=http://localhost:8080/handshake/finalize \
	-session-test-url=http://localhost:8080/session/test