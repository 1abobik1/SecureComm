#!/usr/bin/env bash
set -euo pipefail

KEY_DIR=keys

mkdir -p "$KEY_DIR"
chmod 700 "$KEY_DIR"

# RSA
if [[ ! -f "$KEY_DIR/client_rsa.pem" || ! -f "$KEY_DIR/client_rsa.pub" ]]; then
  echo "Generating RSA keypair…"
  openssl genpkey -algorithm RSA -out "$KEY_DIR/client_rsa.pem" -pkeyopt rsa_keygen_bits:3072
  openssl rsa -in "$KEY_DIR/client_rsa.pem" -pubout -out "$KEY_DIR/client_rsa.pub"
  chmod 600 "$KEY_DIR/client_rsa.pem"
  chmod 644 "$KEY_DIR/client_rsa.pub"
else
  echo "RSA keys exist, skipping"
fi

# ECDSA
if [[ ! -f "$KEY_DIR/client_ecdsa.pem" || ! -f "$KEY_DIR/client_ecdsa.pub" ]]; then
  echo "Generating ECDSA keypair…"
  openssl ecparam -genkey -name prime256v1 -noout -out "$KEY_DIR/client_ecdsa.pem"
  openssl ec -in "$KEY_DIR/client_ecdsa.pem" -pubout -out "$KEY_DIR/client_ecdsa.pub"
  chmod 600 "$KEY_DIR/client_ecdsa.pem"
  chmod 644 "$KEY_DIR/client_ecdsa.pub"
else
  echo "ECDSA keys exist, skipping"
fi