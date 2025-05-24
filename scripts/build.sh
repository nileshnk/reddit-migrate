#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
# The main package is now in cmd/reddit-migrate
MAIN_PACKAGE_PATH="./cmd/reddit-migrate"
# Updated LD_FLAGS with additional optimization and stripping
LD_FLAGS="-X main.Version=${VERSION} -s -w"

# Create a directory for the builds if it doesn't exist
mkdir -p builds

PROJECT_NAME="reddit-migrate" # Hardcoded as per request

# Function to create .app bundle for macOS
create_macos_app_bundle() {
  local arch=$1
  local project_name_param=$2
  local ld_flags_param=$3
  local main_pkg_path_param=$4 # Added main package path parameter
  local output_binary_base="builds/${project_name_param}-macos-${arch}"
  local app_bundle_path="builds/${project_name_param}-macos-${arch}.app"

  echo "Building macOS binary for ${arch} (${project_name_param}) from ${main_pkg_path_param}..."
  # Disable CGO for static binary, add build mode for smaller size
  CGO_ENABLED=0 GOOS=darwin GOARCH=${arch} go build -buildmode=pie -ldflags="${ld_flags_param}" -o "${output_binary_base}" "${main_pkg_path_param}"
  chmod +x "${output_binary_base}"

  echo "Creating .app bundle for macOS (${arch}) at ${app_bundle_path}..."
  mkdir -p "${app_bundle_path}/Contents/MacOS"
  mkdir -p "${app_bundle_path}/Contents/Resources"
  cp "${output_binary_base}" "${app_bundle_path}/Contents/MacOS/${project_name_param}"

  # Create Info.plist with additional keys for better macOS compatibility
  cat > "${app_bundle_path}/Contents/Info.plist" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${project_name_param}</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.${project_name_param}</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
    <key>CFBundleName</key>
    <string>${project_name_param}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
    <key>NSAppTransportSecurity</key>
    <dict>
        <key>NSAllowsArbitraryLoads</key>
        <true/>
    </dict>
</dict>
</plist>
EOL

  # Create entitlements file for code signing
  cat > "${app_bundle_path}/Contents/Resources/entitlements.plist" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>com.apple.security.network.client</key>
    <true/>
    <key>com.apple.security.network.server</key>
    <true/>
</dict>
</plist>
EOL

  # Icon path relative to project root, script is in scripts/
  local root_icon_png="../icon.png"
  if [ -f "${root_icon_png}" ]; then
    local iconset_path="${app_bundle_path}/Contents/Resources/icon.iconset"
    mkdir -p "$iconset_path"
    sips -z 16 16     "${root_icon_png}" --out "$iconset_path/icon_16x16.png" >/dev/null 2>&1
    sips -z 32 32     "${root_icon_png}" --out "$iconset_path/icon_16x16@2x.png" >/dev/null 2>&1
    sips -z 32 32     "${root_icon_png}" --out "$iconset_path/icon_32x32.png" >/dev/null 2>&1
    sips -z 64 64     "${root_icon_png}" --out "$iconset_path/icon_32x32@2x.png" >/dev/null 2>&1
    sips -z 128 128   "${root_icon_png}" --out "$iconset_path/icon_128x128.png" >/dev/null 2>&1
    sips -z 256 256   "${root_icon_png}" --out "$iconset_path/icon_128x128@2x.png" >/dev/null 2>&1
    sips -z 256 256   "${root_icon_png}" --out "$iconset_path/icon_256x256.png" >/dev/null 2>&1
    sips -z 512 512   "${root_icon_png}" --out "$iconset_path/icon_256x256@2x.png" >/dev/null 2>&1
    sips -z 512 512   "${root_icon_png}" --out "$iconset_path/icon_512x512.png" >/dev/null 2>&1
    sips -z 1024 1024 "${root_icon_png}" --out "$iconset_path/icon_512x512@2x.png" >/dev/null 2>&1
    iconutil -c icns "$iconset_path" -o "${app_bundle_path}/Contents/Resources/icon.icns"
    rm -R "$iconset_path"
  else
    echo "${root_icon_png} not found, skipping icon creation for ${app_bundle_path}."
  fi
  
  # Ad-hoc sign the app bundle (this allows it to run on the build machine)
  # For distribution, you'll need a proper Developer ID certificate
  echo "Ad-hoc signing ${app_bundle_path} for local testing..."
  codesign --force --deep --sign - "${app_bundle_path}"
  
  echo "Raw binary ${output_binary_base} is kept for packaging."
  
  # Print instructions for proper signing
  echo ""
  echo "NOTE: For distribution to other machines, you need to:"
  echo "1. Sign with a Developer ID: codesign --force --deep --sign 'Developer ID Application: Your Name' ${app_bundle_path}"
  echo "2. Notarize the app with Apple"
  echo "3. Or instruct users to right-click and select 'Open' to bypass Gatekeeper"
  echo ""
}

package_build() {
  local os_target=$1
  local arch_target=$2
  local project_name_global=$3
  local version_global=$4
  
  local source_artifact_basename="${project_name_global}-${os_target}-${arch_target}"
  local executable_name_in_package="${project_name_global}"

  if [ "${os_target}" = "windows" ]; then
    source_artifact_basename+=".exe"
    executable_name_in_package+=".exe"
  fi

  local temp_staging_dir="builds/staging_temp_${os_target}_${arch_target}"
  local package_content_root="${temp_staging_dir}/${project_name_global}"
  local final_zip_filepath="builds/${project_name_global}-${version_global}-${os_target}-${arch_target}.zip"

  echo "Packaging for ${os_target}-${arch_target} into ${final_zip_filepath}..."

  rm -rf "${temp_staging_dir}" 
  mkdir -p "${package_content_root}"

  if [ "${os_target}" = "macos" ]; then
    echo "Moving ${project_name_global}.app (from builds/${source_artifact_basename}.app) to staging..."
    cp -R "builds/${source_artifact_basename}.app" "${package_content_root}/${project_name_global}.app"
    
    echo "Moving raw macOS binary (from builds/${source_artifact_basename}) to staging/bin/ ..."
    mkdir -p "${package_content_root}/bin"
    cp "builds/${source_artifact_basename}" "${package_content_root}/bin/${project_name_global}"
    
    # Add a README for macOS users
    cat > "${package_content_root}/README_MACOS.txt" <<EOL
Reddit Migrate for macOS

To run the application:

Option 1 - Use the .app bundle:
1. Right-click on ${project_name_global}.app
2. Select "Open" from the context menu
3. Click "Open" in the security dialog
   (This bypasses Gatekeeper for unsigned apps)

Option 2 - Use the command line binary:
1. Open Terminal
2. Navigate to the bin/ directory
3. Run: ./${project_name_global}

If you get a "cannot be opened" error:
- Go to System Preferences > Security & Privacy
- Click "Open Anyway" for this app

For automatic opening without warnings, the app
needs to be signed with an Apple Developer ID.
EOL
  else
    echo "Moving executable (from builds/${source_artifact_basename}) to staging..."
    cp "builds/${source_artifact_basename}" "${package_content_root}/${executable_name_in_package}"
  fi

  # public folder path relative to project root
  local root_public_dir="web"
  if [ -d "${root_public_dir}" ]; then
    echo "Copying ${root_public_dir} folder to staging..."
    cp -R "${root_public_dir}" "${package_content_root}/" # Copies 'web' into 'PROJECT_NAME/web' in staging
    echo "Included 'web' folder in ${final_zip_filepath}"
  else
    echo "'${root_public_dir}' folder not found at $(pwd)/${root_public_dir}, skipping."
  fi

  echo "Creating ZIP archive: ${final_zip_filepath}"
  local zip_filename_only="${project_name_global}-${version_global}-${os_target}-${arch_target}.zip"
  
  if [ "${os_target}" = "macos" ]; then
    # For macOS, create the zip without extended attributes to avoid quarantine issues
    (cd "${temp_staging_dir}" && zip -qr -X "../${zip_filename_only}" "${project_name_global}")
  else
    (cd "${temp_staging_dir}" && zip -qr "../${zip_filename_only}" "${project_name_global}")
  fi
  
  # Clean up original artifacts after packaging
  rm -f "builds/${source_artifact_basename}"
  if [ "${os_target}" = "macos" ]; then
    rm -rf "builds/${source_artifact_basename}.app"
  fi
  
  rm -rf "${temp_staging_dir}"
  echo "Successfully created ${final_zip_filepath}"
}

# Build targets with CGO disabled for all platforms
# Linux
echo "Building for Linux (amd64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-amd64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-linux-amd64"
package_build "linux" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Linux (arm64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-arm64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-linux-arm64"
package_build "linux" "arm64" "$PROJECT_NAME" "$VERSION"

# Windows
echo "Building for Windows (amd64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-amd64.exe" "${MAIN_PACKAGE_PATH}"
package_build "windows" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Windows (arm64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-arm64.exe" "${MAIN_PACKAGE_PATH}"
package_build "windows" "arm64" "$PROJECT_NAME" "$VERSION"

# macOS
create_macos_app_bundle "amd64" "$PROJECT_NAME" "$LD_FLAGS" "$MAIN_PACKAGE_PATH"
package_build "macos" "amd64" "$PROJECT_NAME" "$VERSION"

create_macos_app_bundle "arm64" "$PROJECT_NAME" "$LD_FLAGS" "$MAIN_PACKAGE_PATH"
package_build "macos" "arm64" "$PROJECT_NAME" "$VERSION"

# FreeBSD
echo "Building for FreeBSD (amd64)..."
CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-amd64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-freebsd-amd64"
package_build "freebsd" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for FreeBSD (arm64)..."
CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-arm64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-freebsd-arm64"
package_build "freebsd" "arm64" "$PROJECT_NAME" "$VERSION"

echo "All builds completed. ZIP archives are in the 'builds' directory." 