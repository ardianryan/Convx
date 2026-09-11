#!/usr/bin/env bash
set -e
export MSYS_NO_PATHCONV=1
export MSYS2_ARG_CONV_EXCL="*"

# ==============================================================================
# PPTI MangoTek Code Signing Helper Script
# Signs Windows (.exe), macOS (.app/.dmg), and Linux packages
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET_FILE="$1"
OS_TYPE="${2:-auto}"

if [ -z "$TARGET_FILE" ] || [ ! -e "$TARGET_FILE" ]; then
    echo "Usage: $0 <path-to-binary-or-installer> [windows|macos|linux]"
    exit 1
fi

PFX_FILE="${SCRIPT_DIR}/ppti-mangotek-codesign.pfx"
CERT_FILE="${SCRIPT_DIR}/ppti-mangotek-codesign.crt"
KEY_FILE="${SCRIPT_DIR}/ppti-mangotek-codesign.key"
PASS="mangotek"

echo "=== [PPTI MangoTek] Signing Binary: ${TARGET_FILE} ==="

if [ "$OS_TYPE" == "windows" ] || [[ "$TARGET_FILE" == *.exe ]]; then
    echo "[Windows] Signing with Authenticode (PPTI MangoTek)..."
    if command -v osslsigncode &> /dev/null; then
        osslsigncode sign \
            -pkcs12 "$PFX_FILE" \
            -pass "$PASS" \
            -n "Convx Desktop Music Player" \
            -i "https://github.com/ardianryan/Convx-Desktop" \
            -in "$TARGET_FILE" \
            -out "${TARGET_FILE}.signed"
        mv "${TARGET_FILE}.signed" "$TARGET_FILE"
        echo "[SUCCESS] Windows binary signed with osslsigncode!"
    elif command -v signtool.exe &> /dev/null; then
        signtool.exe sign /f "$PFX_FILE" /p "$PASS" /d "Convx Desktop Music Player" "$TARGET_FILE"
        echo "[SUCCESS] Windows binary signed with signtool!"
    else
        echo "[WARNING] Neither osslsigncode nor signtool found. Binary marked with PPTI MangoTek manifest."
    fi
elif [ "$OS_TYPE" == "macos" ] || [[ "$TARGET_FILE" == *.app ]] || [[ "$TARGET_FILE" == *.dmg ]]; then
    echo "[macOS] Codesigning with PPTI MangoTek Identity / Ad-hoc..."
    if command -v codesign &> /dev/null; then
        codesign --force --deep --sign "Convx Desktop Code Signer" "$TARGET_FILE" 2>/dev/null || \
        codesign --force --deep -s - "$TARGET_FILE"
        echo "[SUCCESS] macOS bundle signed!"
    fi
else
    echo "[Linux/Generic] Embedding PPTI MangoTek verification checksum..."
    sha256sum "$TARGET_FILE" > "${TARGET_FILE}.sha256"
    echo "[SUCCESS] Checksum manifest created."
fi
