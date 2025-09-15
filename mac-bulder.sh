#!/bin/bash

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Создание приложения Bitrix24 Repackager...${NC}"

# Проверяем что бинарный файл существует
if [ ! -f "bitrix-repackager-macos" ]; then
    echo -e "${RED}Ошибка: бинарный файл bitrix-repackager-macos не найден${NC}"
    echo "Сначала выполните: go build -o bitrix-repackager-macos"
    exit 1
fi

# Создаем структуру приложения
APP_NAME="Bitrix24 Repackager.app"
echo -e "${YELLOW}Создаем структуру приложения...${NC}"
mkdir -p "$APP_NAME/Contents/MacOS"
mkdir -p "$APP_NAME/Contents/Resources"

# Копируем бинарный файл
echo -e "${YELLOW}Копируем бинарный файл...${NC}"
cp bitrix-repackager-macos "$APP_NAME/Contents/MacOS/"

# Создаем скрипт запуска
echo -e "${YELLOW}Создаем скрипт запуска...${NC}"
cat > "$APP_NAME/Contents/MacOS/bitrix-repackager" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
exec "./bitrix-repackager-macos"
EOF

chmod +x "$APP_NAME/Contents/MacOS/bitrix-repackager"
chmod +x "$APP_NAME/Contents/MacOS/bitrix-repackager-macos"

# Создаем Info.plist
echo -e "${YELLOW}Создаем Info.plist...${NC}"
cat > "$APP_NAME/Contents/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>bitrix-repackager</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>bitrix24ad.repackager</string>
    <key>CFBundleName</key>
    <string>Bitrix24 Repackager</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# Генерируем AppIcon.icns из одного PNG (например, icon.png 512x512)
echo -e "${YELLOW}Генерируем AppIcon.icns из icon.png...${NC}"

if [ ! -f "icon.png" ]; then
    echo -e "${RED}Ошибка: файл icon.png не найден!${NC}"
    echo "Поместите PNG-иконку (рекомендуется 512x512) с именем icon.png в текущую директорию."
    exit 1
fi

# Проверим, что это PNG
if ! file icon.png | grep -q "PNG image data"; then
    echo -e "${RED}Ошибка: icon.png не является валидным PNG-файлом${NC}"
    exit 1
fi

# Создаём временную папку
TEMP_ICONSET="temp.iconset"
rm -rf "$TEMP_ICONSET"
mkdir -p "$TEMP_ICONSET"

# Копируем исходник
cp icon.png "$TEMP_ICONSET/icon_512x512.png"

# Генерируем все размеры
# shellcheck disable=SC2164
cd "$TEMP_ICONSET"

sips -z 16  16  icon_512x512.png --out icon_16x16.png >/dev/null
sips -z 32  32  icon_512x512.png --out icon_16x16@2x.png >/dev/null
sips -z 32  32  icon_512x512.png --out icon_32x32.png >/dev/null
sips -z 64  64  icon_512x512.png --out icon_32x32@2x.png >/dev/null
sips -z 128 128 icon_512x512.png --out icon_128x128.png >/dev/null
sips -z 256 256 icon_512x512.png --out icon_128x128@2x.png >/dev/null
sips -z 256 256 icon_512x512.png --out icon_256x256.png >/dev/null
sips -z 512 512 icon_512x512.png --out icon_256x256@2x.png >/dev/null
sips -z 512 512 icon_512x512.png --out icon_512x512.png >/dev/null
sips -z 1024 1024 icon_512x512.png --out icon_512x512@2x.png >/dev/null

# Создаём Contents.json
cat > Contents.json << 'EOF'
{
  "images": [
    { "size": "16x16",   "idiom": "mac", "filename": "icon_16x16.png",     "scale": "1x" },
    { "size": "16x16",   "idiom": "mac", "filename": "icon_16x16@2x.png",   "scale": "2x" },
    { "size": "32x32",   "idiom": "mac", "filename": "icon_32x32.png",     "scale": "1x" },
    { "size": "32x32",   "idiom": "mac", "filename": "icon_32x32@2x.png",   "scale": "2x" },
    { "size": "128x128", "idiom": "mac", "filename": "icon_128x128.png",   "scale": "1x" },
    { "size": "128x128", "idiom": "mac", "filename": "icon_128x128@2x.png", "scale": "2x" },
    { "size": "256x256", "idiom": "mac", "filename": "icon_256x256.png",   "scale": "1x" },
    { "size": "256x256", "idiom": "mac", "filename": "icon_256x256@2x.png", "scale": "2x" },
    { "size": "512x512", "idiom": "mac", "filename": "icon_512x512.png",   "scale": "1x" },
    { "size": "512x512", "idiom": "mac", "filename": "icon_512x512@2x.png", "scale": "2x" }
  ],
  "info": {
    "version": 1,
    "author": "generated"
  }
}
EOF

# shellcheck disable=SC2103
cd ..

# Конвертируем в .icns
if iconutil -c icns "$TEMP_ICONSET" -o AppIcon.icns; then
    cp AppIcon.icns "$APP_NAME/Contents/Resources/"
    echo -e "${GREEN}✅ AppIcon.icns успешно создан и скопирован!${NC}"
else
    echo -e "${RED}❌ Не удалось создать AppIcon.icns${NC}"
    exit 1
fi

# Убираем временные файлы
rm -rf "$TEMP_ICONSET" AppIcon.icns
