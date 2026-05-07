#!/bin/bash
APP_NAME="Helper"

echo "Compilando e compactando..."

# Limpa dist anterior
rm -rf dist
mkdir -p dist

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${APP_NAME}_windows_amd64.exe" .
zip -j "dist/${APP_NAME}_windows_amd64.zip" "dist/${APP_NAME}_windows_amd64.exe"
rm "dist/${APP_NAME}_windows_amd64.exe"  # Remove o cru

# Linux
for arch in amd64 arm64; do
    GOOS=linux GOARCH=$arch go build -ldflags="-s -w" -o "dist/${APP_NAME}_linux_$arch" .
    tar -czf "dist/${APP_NAME}_linux_$arch.tar.gz" -C dist "${APP_NAME}_linux_$arch"
    rm "dist/${APP_NAME}_linux_$arch"  # Remove o cru
done

# macOS
for arch in amd64 arm64; do
    GOOS=darwin GOARCH=$arch go build -ldflags="-s -w" -o "dist/${APP_NAME}_darwin_$arch" .
    tar -czf "dist/${APP_NAME}_darwin_$arch.tar.gz" -C dist "${APP_NAME}_darwin_$arch"
    rm "dist/${APP_NAME}_darwin_$arch"  # Remove o cru
done

# Gera checksums SÓ dos compactados
cd dist
sha256sum *.tar.gz *.zip > ${APP_NAME}_checksums.txt

echo "✅ Gerado:"
ls -lh