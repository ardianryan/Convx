#!/usr/bin/env bash
set -e

# ==============================================================================
# PPTI MangoTek Certificate Trust Installer for macOS & Linux
# Installs PPTI MangoTek Root CA into macOS System Keychain / Linux CA store
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_FILE="${SCRIPT_DIR}/ppti-mangotek-rootca.crt"

if [ ! -f "$CERT_FILE" ]; then
    echo "[ERROR] Certificate file not found at '$CERT_FILE'"
    exit 1
fi

echo "=== [PPTI MangoTek] Registering Trusted Root Certificate ==="

if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "[macOS] Adding PPTI MangoTek Root CA to System Keychain..."
    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$CERT_FILE" || true
    echo "[macOS] Clearing quarantine flags & re-signing app bundle..."
    if [ -d "${SCRIPT_DIR}/Convx.app" ]; then
        xattr -cr "${SCRIPT_DIR}/Convx.app" 2>/dev/null || true
        codesign --force --deep -s - "${SCRIPT_DIR}/Convx.app" 2>/dev/null || true
    elif [ -d "${SCRIPT_DIR}/convx-desktop.app" ]; then
        xattr -cr "${SCRIPT_DIR}/convx-desktop.app" 2>/dev/null || true
        codesign --force --deep -s - "${SCRIPT_DIR}/convx-desktop.app" 2>/dev/null || true
    fi
    echo "[SUCCESS] PPTI MangoTek Root CA & macOS App Trust updated!"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "[Linux] Adding PPTI MangoTek Root CA to system CA store..."
    if [ -d "/usr/local/share/ca-certificates" ]; then
        sudo cp "$CERT_FILE" /usr/local/share/ca-certificates/ppti-mangotek-rootca.crt
        sudo update-ca-certificates
    elif [ -d "/etc/pki/ca-trust/source/anchors" ]; then
        sudo cp "$CERT_FILE" /etc/pki/ca-trust/source/anchors/ppti-mangotek-rootca.crt
        sudo update-ca-trust
    else
        echo "[WARNING] Unknown CA store directory. Please copy '$CERT_FILE' to your distribution's CA directory."
        exit 1
    fi
    echo "[SUCCESS] PPTI MangoTek Root CA added to Linux System Store!"
else
    echo "[ERROR] Unsupported operating system: $OSTYPE"
    exit 1
fi
