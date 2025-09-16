#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Создание Windows-приложения Bitrix24 Bad Advice Repackager...${NC}"

# Проверяем, что бинарный файл существует
if [ ! -f "b24ad-repackager-windows.exe" ]; then
    echo -e "${RED}Ошибка: бинарный файл b24ad-repackager-windows.exe не найден${NC}"
    echo "Сначала выполните: GOOS=windows GOARCH=amd64 go build -o b24ad-repackager-windows.exe"
    exit 1
fi

APP_DIR="Bitrix24AD-Repackager-win"
DIST_DIR="Bitrix24AD-Repackager-win-dist"
ZIP_NAME="Bitrix24AD-Repackager-win.zip"

echo -e "${YELLOW}Создаём структуру приложения...${NC}"
rm -rf "$APP_DIR" "$DIST_DIR" "$ZIP_NAME"
mkdir -p "$APP_DIR"
mkdir -p "$DIST_DIR"

# Копируем бинарник
echo -e "${YELLOW}Копируем бинарный файл...${NC}"
cp b24ad-repackager-windows.exe "$APP_DIR/"

# Копируем иконку, если есть
if [ -f "icon.ico" ]; then
    echo -e "${YELLOW}Копируем иконку...${NC}"
    cp icon.ico "$APP_DIR/"
else
    echo -e "${YELLOW}⚠️  Файл icon.ico не найден, будет использована стандартная иконка Windows${NC}"
fi

# Создаём ярлык (lnk) через powershell, если возможно
echo -e "${YELLOW}Создаём ярлык Bitrix24AD Repackager.lnk...${NC}"
cat > "$APP_DIR/create-shortcut.ps1" << 'EOF'
$WshShell = New-Object -ComObject WScript.Shell
$Shortcut = $WshShell.CreateShortcut("$PSScriptRoot\Bitrix24AD Repackager.lnk")
$Shortcut.TargetPath = "$PSScriptRoot\b24ad-repackager-windows.exe"
$Shortcut.WorkingDirectory = "$PSScriptRoot"
if (Test-Path "$PSScriptRoot\icon.ico") {
    $Shortcut.IconLocation = "$PSScriptRoot\icon.ico"
}
$Shortcut.Save()
EOF

# Запускаем PowerShell для создания ярлыка (если на Linux/macOS — пропускаем)
if command -v powershell.exe >/dev/null 2>&1; then
    (cd "$APP_DIR" && powershell.exe -ExecutionPolicy Bypass -File create-shortcut.ps1)
elif command -v pwsh >/dev/null 2>&1; then
    (cd "$APP_DIR" && pwsh -ExecutionPolicy Bypass -File create-shortcut.ps1)
else
    echo -e "${YELLOW}PowerShell не найден, ярлык не будет создан автоматически. Создайте вручную на Windows.${NC}"
fi

rm -f "$APP_DIR/create-shortcut.ps1"

# Копируем README, если есть
if [ -f "README.md" ]; then
    cp README.md "$APP_DIR/"
fi

# Переносим всё в папку для архивации
mv "$APP_DIR" "$DIST_DIR/"

# Создаём zip-архив
echo -e "${YELLOW}Создаём архив $ZIP_NAME...${NC}"
(cd "$DIST_DIR" && zip -r "../$ZIP_NAME" .)

echo -e "${GREEN}✅ Windows-дистрибутив успешно собран: $ZIP_NAME${NC}"