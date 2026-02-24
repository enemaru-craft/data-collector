#!/usr/bin/env bash
set -euo pipefail

# MQTTS用の自己署名証明書を生成
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
CERT_DIR="$SCRIPT_DIR/../mosquitto/certs"

mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

# CA証明書
if [ ! -f ca.key ]; then
  echo "[INFO] Generating CA key and certificate..."
  openssl genrsa -out ca.key 2048
  openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
    -subj "/CN=LocalMQTT-CA" -out ca.crt
  echo "[OK] CA certificate created"
fi

# サーバー証明書
if [ ! -f server.key ]; then
  echo "[INFO] Generating server key and certificate..."
  openssl genrsa -out server.key 2048
  openssl req -new -key server.key -subj "/CN=mosquitto" -out server.csr

  # SAN (Subject Alternative Name) を含める
  cat > server.ext << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = mosquitto
DNS.2 = localhost
DNS.3 = data-collector-mosquitto
IP.1 = 127.0.0.1
EOF

  openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out server.crt -days 3650 -sha256 -extfile server.ext
  rm -f server.csr server.ext
  echo "[OK] Server certificate created"
fi

# クライアント証明書 (オプション、相互TLS用)
if [ ! -f client.key ]; then
  echo "[INFO] Generating client key and certificate..."
  openssl genrsa -out client.key 2048
  openssl req -new -key client.key -subj "/CN=mqtt-client" -out client.csr
  openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
    -out client.crt -days 3650 -sha256
  rm -f client.csr
  echo "[OK] Client certificate created"
fi

echo ""
echo "=========================================="
echo "Certificates generated in: $CERT_DIR"
echo ""
echo "Files:"
echo "  ca.crt      - CA certificate (trust this on clients)"
echo "  server.crt  - Server certificate"
echo "  server.key  - Server private key"
echo "  client.crt  - Client certificate (for mutual TLS)"
echo "  client.key  - Client private key"
echo "=========================================="
