#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
# The main package is now in cmd/reddit-migrate
MAIN_PACKAGE_PATH="./cmd/reddit-migrate"
LD_FLAGS="-X main.Version=${VERSION}"

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
  GOOS=darwin GOARCH=${arch} go build -ldflags="${ld_flags_param}" -o "${output_binary_base}" "${main_pkg_path_param}"
  chmod +x "${output_binary_base}"

  echo "Creating .app bundle for macOS (${arch}) at ${app_bundle_path}..."
  mkdir -p "${app_bundle_path}/Contents/MacOS"
  mkdir -p "${app_bundle_path}/Contents/Resources"
  cp "${output_binary_base}" "${app_bundle_path}/Contents/MacOS/${project_name_param}"

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
</dict>
</plist>
EOL

  # Icon path relative to project root, script is in scripts/
  local root_icon_png="../icon.png"
  if [ -f "${root_icon_png}" ]; then
    local iconset_path="${app_bundle_path}/Contents/Resources/icon.iconset"
    mkdir -p "$iconset_path"
    sips -z 16 16     "${root_icon_png}" --out "$iconset_path/icon_16x16.png"
    sips -z 32 32     "${root_icon_png}" --out "$iconset_path/icon_16x16@2x.png"
    sips -z 32 32     "${root_icon_png}" --out "$iconset_path/icon_32x32.png"
    sips -z 64 64     "${root_icon_png}" --out "$iconset_path/icon_32x32@2x.png"
    sips -z 128 128   "${root_icon_png}" --out "$iconset_path/icon_128x128.png"
    sips -z 256 256   "${root_icon_png}" --out "$iconset_path/icon_128x128@2x.png"
    sips -z 256 256   "${root_icon_png}" --out "$iconset_path/icon_256x256.png"
    sips -z 512 512   "${root_icon_png}" --out "$iconset_path/icon_256x256@2x.png"
    sips -z 512 512   "${root_icon_png}" --out "$iconset_path/icon_512x512.png"
    sips -z 1024 1024 "${root_icon_png}" --out "$iconset_path/icon_512x512@2x.png"
    iconutil -c icns "$iconset_path" -o "${app_bundle_path}/Contents/Resources/icon.icns"
    rm -R "$iconset_path"
  else
    echo "${root_icon_png} not found, skipping icon creation for ${app_bundle_path}."
  fi
  
  echo "Raw binary ${output_binary_base} is kept for packaging."
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
    mv "builds/${source_artifact_basename}.app" "${package_content_root}/${project_name_global}.app"
    
    echo "Moving raw macOS binary (from builds/${source_artifact_basename}) to staging/bin/ ..."
    mkdir -p "${package_content_root}/bin"
    mv "builds/${source_artifact_basename}" "${package_content_root}/bin/${project_name_global}"
  else
    echo "Moving executable (from builds/${source_artifact_basename}) to staging..."
    mv "builds/${source_artifact_basename}" "${package_content_root}/${executable_name_in_package}"
  fi

  # public folder path relative to project root
  local root_public_dir="../public"
  if [ -d "${root_public_dir}" ]; then
    echo "Copying ${root_public_dir} folder to staging..."
    cp -R "${root_public_dir}" "${package_content_root}/" # Copies 'public' into 'PROJECT_NAME/public' in staging
    echo "Included 'public' folder in ${final_zip_filepath}"
  else
    echo "'${root_public_dir}' folder not found at $(pwd)/${root_public_dir}, skipping."
  fi

  echo "Creating ZIP archive: ${final_zip_filepath}"
  local zip_filename_only="${project_name_global}-${version_global}-${os_target}-${arch_target}.zip"
  (cd "${temp_staging_dir}" && zip -qr "../${zip_filename_only}" "${project_name_global}")
  
  rm -rf "${temp_staging_dir}"
  echo "Successfully created ${final_zip_filepath}"
}

# Build targets
# Linux
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-amd64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-linux-amd64"
package_build "linux" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-arm64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-linux-arm64"
package_build "linux" "arm64" "$PROJECT_NAME" "$VERSION"

# Windows
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-amd64.exe" "${MAIN_PACKAGE_PATH}"
package_build "windows" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-arm64.exe" "${MAIN_PACKAGE_PATH}"
package_build "windows" "arm64" "$PROJECT_NAME" "$VERSION"

# macOS
create_macos_app_bundle "amd64" "$PROJECT_NAME" "$LD_FLAGS" "$MAIN_PACKAGE_PATH"
package_build "macos" "amd64" "$PROJECT_NAME" "$VERSION"

create_macos_app_bundle "arm64" "$PROJECT_NAME" "$LD_FLAGS" "$MAIN_PACKAGE_PATH"
package_build "macos" "arm64" "$PROJECT_NAME" "$VERSION"

# FreeBSD
echo "Building for FreeBSD (amd64)..."
GOOS=freebsd GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-amd64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-freebsd-amd64"
package_build "freebsd" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for FreeBSD (arm64)..."
GOOS=freebsd GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-arm64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-freebsd-arm64"
package_build "freebsd" "arm64" "$PROJECT_NAME" "$VERSION"

echo "All builds completed. ZIP archives are in the 'builds' directory." 