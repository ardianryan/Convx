#!/usr/bin/env bash
set -e

# ==============================================================================
# OpenSSL Code Signing Certificate Generator
# Organization: PPTI MangoTek
# Description: Generates Root CA & Code Signing certificate for Convx Desktop
# ==============================================================================

CERTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$CERTS_DIR"

ORG_NAME="PPTI MangoTek"
COUNTRY="ID"
STATE="Jakarta"
LOCALITY="Jakarta"
ROOT_CN="PPTI MangoTek Root CA"
CODESIGN_CN="Convx Desktop Code Signer"
PASS="mangotek"

echo "=== [PPTI MangoTek] Generating OpenSSL Certificates ==="

# 1. Generate Root CA Private Key & Self-Signed Certificate
echo "[1/4] Generating Root CA Key and Certificate..."
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout ppti-mangotek-rootca.key \
  -out ppti-mangotek-rootca.crt \
  -days 3650 \
  -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG_NAME}/CN=${ROOT_CN}" \
  -addext "basicConstraints=critical,CA:TRUE" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"

# 2. Generate Code Signing CSR & Private Key
echo "[2/4] Generating Code Signing Private Key & CSR..."
openssl req -newkey rsa:2048 -nodes \
  -keyout ppti-mangotek-codesign.key \
  -out ppti-mangotek-codesign.csr \
  -subj "/C=${COUNTRY}/ST=${STATE}/L=${LOCALITY}/O=${ORG_NAME}/CN=${CODESIGN_CN}"

# 3. Create OpenSSL ext file for Code Signing
cat << EOF > codesign.ext
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = critical, digitalSignature
extendedKeyUsage = codeSigning
EOF

# 4. Sign Code Signing Certificate with Root CA
echo "[3/4] Signing Code Signing Certificate with Root CA..."
openssl x509 -req \
  -in ppti-mangotek-codesign.csr \
  -CA ppti-mangotek-rootca.crt \
  -CAkey ppti-mangotek-rootca.key \
  -CAcreateserial \
  -out ppti-mangotek-codesign.crt \
  -days 1825 \
  -extfile codesign.ext

# 5. Export to PKCS#12 (.pfx / .p12) for Windows Authenticode signing
echo "[4/4] Bundling into PKCS#12 (.pfx) for Authenticode signing..."
openssl pkcs12 -export \
  -out ppti-mangotek-codesign.pfx \
  -inkey ppti-mangotek-codesign.key \
  -in ppti-mangotek-codesign.crt \
  -certfile ppti-mangotek-rootca.crt \
  -passout pass:${PASS}

# Clean temporary files
rm -f ppti-mangotek-codesign.csr codesign.ext ppti-mangotek-rootca.srl

echo ""
echo "=== [SUCCESS] Certificates generated in ${CERTS_DIR} ==="
echo "Root CA Cert     : ppti-mangotek-rootca.crt"
echo "CodeSign Cert    : ppti-mangotek-codesign.crt"
echo "PFX Package      : ppti-mangotek-codesign.pfx (Password: ${PASS})"
echo "Organization Name: ${ORG_NAME}"
