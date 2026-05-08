#!/bin/bash
APP_NAME="Helper"

echo "Compilando e compactando..."

# Limpa dist anterior
rm -rf dist
mkdir -p dist

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o "dist/${APP_NAME}_windows_amd64.exe" .

# Linux
for arch in amd64 arm64; do
    GOOS=linux GOARCH=$arch go build -ldflags="-s -w" -o "dist/${APP_NAME}_linux_$arch" .
done

# macOS
for arch in amd64 arm64; do
    GOOS=darwin GOARCH=$arch go build -ldflags="-s -w" -o "dist/${APP_NAME}_darwin_$arch" .
done

echo "✅ Gerado"