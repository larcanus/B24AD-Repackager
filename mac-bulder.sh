#!/bin/bash

# Создаем структуру приложения
mkdir -p "Bitrix24 Repackager.app/Contents/MacOS"
mkdir -p "Bitrix24 Repackager.app/Contents/Resources"

# Копируем бинарный файл
cp bitrix-repackager-macos "Bitrix24 Repackager.app/Contents/MacOS/"

# Создаем скрипт запуска
cat > "Bitrix24 Repackager.app/Contents/MacOS/bitrix-repackager" << 'EOF'
#!/bin/bash
cd "$(dirname "$0")"
exec "./bitrix-repackager-macos"
EOF

chmod +x "Bitrix24 Repackager.app/Contents/MacOS/bitrix-repackager"

# Создаем Info.plist
cat > "Bitrix24 Repackager.app/Contents/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>bitrix-repackager</string>
    <key>CFBundleIdentifier</key>
    <string>com.bitrix24.repackager</string>
    <key>CFBundleName</key>
    <string>Bitrix24 Repackager</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <false/>
</dict>
</plist>
EOF

echo "Приложение создано: Bitrix24 Repackager.app"
echo "Можно запускать двойным кликом!"
