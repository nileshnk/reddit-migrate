#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
LD_FLAGS="-X main.Version=${VERSION}"

# Create a directory for the builds if it doesn't exist
mkdir -p builds

# Get the project name from go.mod (optional, assumes module name is the desired executable name)
# Or set a default project name
PROJECT_NAME=$(grep '^module' go.mod | awk '{print $2}' | sed 's|.*/||')
if [ -z "$PROJECT_NAME" ]; then
  PROJECT_NAME="reddit-migrate"
fi

echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-amd64" .

echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-amd64.exe" .

echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-macos-amd64" .
chmod +x "builds/${PROJECT_NAME}-macos-amd64"

# Create .app bundle for macOS (amd64)
APP_NAME_AMD64="builds/${PROJECT_NAME}-macos-amd64.app"
mkdir -p "${APP_NAME_AMD64}/Contents/MacOS"
mkdir -p "${APP_NAME_AMD64}/Contents/Resources"
cp "builds/${PROJECT_NAME}-macos-amd64" "${APP_NAME_AMD64}/Contents/MacOS/${PROJECT_NAME}"

# Create Info.plist for amd64
cat > "${APP_NAME_AMD64}/Contents/Info.plist" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${PROJECT_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.${PROJECT_NAME}</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOL

# Create icon.icns for amd64 if icon.png exists
if [ -f "icon.png" ]; then
  ICONSET_AMD64="${APP_NAME_AMD64}/Contents/Resources/icon.iconset"
  mkdir -p "$ICONSET_AMD64"
  sips -z 16 16     icon.png --out "$ICONSET_AMD64/icon_16x16.png"
  sips -z 32 32     icon.png --out "$ICONSET_AMD64/icon_16x16@2x.png"
  sips -z 32 32     icon.png --out "$ICONSET_AMD64/icon_32x32.png"
  sips -z 64 64     icon.png --out "$ICONSET_AMD64/icon_32x32@2x.png"
  sips -z 128 128   icon.png --out "$ICONSET_AMD64/icon_128x128.png"
  sips -z 256 256   icon.png --out "$ICONSET_AMD64/icon_128x128@2x.png"
  sips -z 256 256   icon.png --out "$ICONSET_AMD64/icon_256x256.png"
  sips -z 512 512   icon.png --out "$ICONSET_AMD64/icon_256x256@2x.png"
  sips -z 512 512   icon.png --out "$ICONSET_AMD64/icon_512x512.png"
  sips -z 1024 1024 icon.png --out "$ICONSET_AMD64/icon_512x512@2x.png"
  iconutil -c icns "$ICONSET_AMD64" -o "${APP_NAME_AMD64}/Contents/Resources/icon.icns"
  rm -R "$ICONSET_AMD64"
fi

echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-macos-arm64" .
chmod +x "builds/${PROJECT_NAME}-macos-arm64"

# Create .app bundle for macOS (arm64)
APP_NAME_ARM64="builds/${PROJECT_NAME}-macos-arm64.app"
mkdir -p "${APP_NAME_ARM64}/Contents/MacOS"
mkdir -p "${APP_NAME_ARM64}/Contents/Resources"
cp "builds/${PROJECT_NAME}-macos-arm64" "${APP_NAME_ARM64}/Contents/MacOS/${PROJECT_NAME}"

# Create Info.plist for arm64
cat > "${APP_NAME_ARM64}/Contents/Info.plist" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${PROJECT_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.${PROJECT_NAME}</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOL

# Create icon.icns for arm64 if icon.png exists
if [ -f "icon.png" ]; then
  ICONSET_ARM64="${APP_NAME_ARM64}/Contents/Resources/icon.iconset"
  mkdir -p "$ICONSET_ARM64"
  sips -z 16 16     icon.png --out "$ICONSET_ARM64/icon_16x16.png"
  sips -z 32 32     icon.png --out "$ICONSET_ARM64/icon_16x16@2x.png"
  sips -z 32 32     icon.png --out "$ICONSET_ARM64/icon_32x32.png"
  sips -z 64 64     icon.png --out "$ICONSET_ARM64/icon_32x32@2x.png"
  sips -z 128 128   icon.png --out "$ICONSET_ARM64/icon_128x128.png"
  sips -z 256 256   icon.png --out "$ICONSET_ARM64/icon_128x128@2x.png"
  sips -z 256 256   icon.png --out "$ICONSET_ARM64/icon_256x256.png"
  sips -z 512 512   icon.png --out "$ICONSET_ARM64/icon_256x256@2x.png"
  sips -z 512 512   icon.png --out "$ICONSET_ARM64/icon_512x512.png"
  sips -z 1024 1024 icon.png --out "$ICONSET_ARM64/icon_512x512@2x.png"
  iconutil -c icns "$ICONSET_ARM64" -o "${APP_NAME_ARM64}/Contents/Resources/icon.icns"
  rm -R "$ICONSET_ARM64"
fi

echo "Building for FreeBSD (amd64)..."
GOOS=freebsd GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-amd64" .

echo "Building for FreeBSD (arm64)..."
GOOS=freebsd GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-arm64" .

# Make all Unix-like builds executable
chmod +x builds/${PROJECT_NAME}-linux-amd64
chmod +x builds/${PROJECT_NAME}-macos-*
chmod +x builds/${PROJECT_NAME}-freebsd-*

echo "Builds completed. Executables are in the 'builds' directory." 