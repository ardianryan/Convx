@echo off
:: ==============================================================================
:: PPTI MangoTek Certificate Trust Installer for Windows
:: Installs PPTI MangoTek Root CA into Windows Trusted Root Certification Authorities
:: ==============================================================================

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo [ERROR] Administrative privileges required.
    echo Please right-click this script and select "Run as Administrator".
    pause
    exit /b 1
)

set "CERT_FILE=%~dp0ppti-mangotek-rootca.crt"

if not exist "%CERT_FILE%" (
    echo [ERROR] Certificate file not found at "%CERT_FILE%"
    pause
    exit /b 1
)

echo [PPTI MangoTek] Registering Trusted Root Certificate...
certutil -addstore -f "Root" "%CERT_FILE%"

if %errorLevel% equ 0 (
    echo [SUCCESS] PPTI MangoTek Root CA successfully installed into Windows Trusted Root Store!
    echo All Convx Desktop applications signed by PPTI MangoTek are now verified.
) else (
    echo [ERROR] Failed to register certificate.
)

pause
