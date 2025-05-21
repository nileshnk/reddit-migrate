#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
LD_FLAGS="-X main.Version=${VERSION}"

# Create a directory for the builds if it doesn't exist
mkdir -p builds

PROJECT_NAME=$(grep '^module' go.mod | awk '{print $2}' | sed 's|.*/||')
if [ -z "$PROJECT_NAME" ]; then
  PROJECT_NAME="reddit-migrate"
fi

# Function to create .app bundle for macOS
create_macos_app_bundle() {
  local arch=$1
  local project_name_param=$2 # Renamed to avoid conflict with global PROJECT_NAME
  local ld_flags_param=$3   # Renamed for clarity
  local output_binary_base="builds/${project_name_param}-macos-${arch}"
  local app_bundle_path="builds/${project_name_param}-macos-${arch}.app"

  echo "Building macOS binary for ${arch} (${project_name_param})..."
  GOOS=darwin GOARCH=${arch} go build -ldflags="${ld_flags_param}" -o "${output_binary_base}" .
  chmod +x "${output_binary_base}" # Make the raw binary executable

  echo "Creating .app bundle for macOS (${arch}) at ${app_bundle_path}..."
  mkdir -p "${app_bundle_path}/Contents/MacOS"
  mkdir -p "${app_bundle_path}/Contents/Resources"
  # Copy the raw binary into the .app bundle, naming it as project_name_param inside
  cp "${output_binary_base}" "${app_bundle_path}/Contents/MacOS/${project_name_param}"

  # Create Info.plist
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
    <key>LSUIElement</key> # Assuming it's a UI element, adjust if not
    <true/>
    <key>CFBundleName</key>
    <string>${project_name_param}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string> <!-- Global VERSION is fine here -->
    <key>CFBundleVersion</key>
    <string>${VERSION}</string> <!-- Global VERSION is fine here -->
</dict>
</plist>
EOL

  # Create icon.icns if icon.png exists
  if [ -f "icon.png" ]; then
    local iconset_path="${app_bundle_path}/Contents/Resources/icon.iconset"
    mkdir -p "$iconset_path"
    sips -z 16 16     icon.png --out "$iconset_path/icon_16x16.png"
    sips -z 32 32     icon.png --out "$iconset_path/icon_16x16@2x.png"
    sips -z 32 32     icon.png --out "$iconset_path/icon_32x32.png"
    sips -z 64 64     icon.png --out "$iconset_path/icon_32x32@2x.png"
    sips -z 128 128   icon.png --out "$iconset_path/icon_128x128.png"
    sips -z 256 256   icon.png --out "$iconset_path/icon_128x128@2x.png"
    sips -z 256 256   icon.png --out "$iconset_path/icon_256x256.png"
    sips -z 512 512   icon.png --out "$iconset_path/icon_256x256@2x.png"
    sips -z 512 512   icon.png --out "$iconset_path/icon_512x512.png"
    sips -z 1024 1024 icon.png --out "$iconset_path/icon_512x512@2x.png"
    iconutil -c icns "$iconset_path" -o "${app_bundle_path}/Contents/Resources/icon.icns"
    rm -R "$iconset_path"
  else
    echo "icon.png not found, skipping icon creation for ${app_bundle_path}."
  fi
  
  # DO NOT remove the output_binary_base here, as we want to package it too.
  # rm "${output_binary_base}" # This line was removed
  echo "Raw binary ${output_binary_base} is kept for packaging."
}

# Function to package build artifacts into a zip
package_build() {
  local os_target=$1
  local arch_target=$2
  local project_name_global=$3 # Using global PROJECT_NAME passed as param
  local version_global=$4   # Using global VERSION passed as param
  
  # This is the name of the actual file/directory artifact produced by the go build or create_macos_app_bundle in the 'builds/' directory
  local source_artifact_basename="${project_name_global}-${os_target}-${arch_target}"
  # This is how the main executable will be named inside the final package structure (e.g. in root or in bin/)
  local executable_name_in_package="${project_name_global}"

  if [ "${os_target}" = "windows" ]; then
    source_artifact_basename+=".exe" # Go build adds .exe suffix for windows artifacts in 'builds/'
    executable_name_in_package+=".exe" # The name inside the zip will also have .exe
  fi

  local temp_staging_dir="builds/staging_temp_${os_target}_${arch_target}" # Unique staging dir per target
  local package_content_root="${temp_staging_dir}/${project_name_global}"   # This will be the single root folder inside the zip (e.g. "reddit-migrate/")
  local final_zip_filepath="builds/${project_name_global}-${version_global}-${os_target}-${arch_target}.zip"

  echo "Packaging for ${os_target}-${arch_target} into ${final_zip_filepath}..."

  rm -rf "${temp_staging_dir}" 
  mkdir -p "${package_content_root}"

  if [ "${os_target}" = "macos" ]; then
    # macOS: expects .app bundle and the raw binary
    # source_artifact_basename is ${project_name_global}-macos-${arch_target}

    echo "Moving ${project_name_global}.app (from builds/${source_artifact_basename}.app) to staging..."
    mv "builds/${source_artifact_basename}.app" "${package_content_root}/${project_name_global}.app"
    
    echo "Moving raw macOS binary (from builds/${source_artifact_basename}) to staging/bin/ ..."
    mkdir -p "${package_content_root}/bin"
    mv "builds/${source_artifact_basename}" "${package_content_root}/bin/${project_name_global}" # Raw binary named 'project_name_global' inside 'bin/'
  else
    # Linux, Windows, FreeBSD: just the executable in the root of package_content_root
    echo "Moving executable (from builds/${source_artifact_basename}) to staging..."
    mv "builds/${source_artifact_basename}" "${package_content_root}/${executable_name_in_package}"
  fi

  # Copy public folder if it exists
  if [ -d "public" ]; then
    echo "Copying public folder to staging..."
    cp -R "public" "${package_content_root}/" # Copies 'public' into 'PROJECT_NAME/public' in staging
    echo "Included 'public' folder in ${final_zip_filepath}"
  else
    echo "'public' folder not found at $(pwd)/public, skipping."
  fi

  echo "Creating ZIP archive: ${final_zip_filepath}"
  local zip_filename_only="${project_name_global}-${version_global}-${os_target}-${arch_target}.zip"
  # We cd into temp_staging_dir, the zip content is the 'project_name_global' folder, 
  # and output path for zip is '../<zip_filename_only>' relative to temp_staging_dir (i.e. into 'builds/')
  (cd "${temp_staging_dir}" && zip -qr "../${zip_filename_only}" "${project_name_global}")
  
  # Clean up temporary staging directory
  rm -rf "${temp_staging_dir}"
  echo "Successfully created ${final_zip_filepath}"
}

# Build targets

# Linux
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-amd64" .
chmod +x "builds/${PROJECT_NAME}-linux-amd64"
package_build "linux" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-linux-arm64" .
chmod +x "builds/${PROJECT_NAME}-linux-arm64"
package_build "linux" "arm64" "$PROJECT_NAME" "$VERSION"

# Windows
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-amd64.exe" .
package_build "windows" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for Windows (arm64)..."
GOOS=windows GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-windows-arm64.exe" .
package_build "windows" "arm64" "$PROJECT_NAME" "$VERSION"

# macOS
# For macOS, create_macos_app_bundle will produce:
# builds/${PROJECT_NAME}-macos-amd64 (raw executable)
# builds/${PROJECT_NAME}-macos-amd64.app (app bundle)
create_macos_app_bundle "amd64" "$PROJECT_NAME" "$LD_FLAGS"
package_build "macos" "amd64" "$PROJECT_NAME" "$VERSION"

create_macos_app_bundle "arm64" "$PROJECT_NAME" "$LD_FLAGS"
package_build "macos" "arm64" "$PROJECT_NAME" "$VERSION"

# FreeBSD
echo "Building for FreeBSD (amd64)..."
GOOS=freebsd GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-amd64" .
chmod +x "builds/${PROJECT_NAME}-freebsd-amd64"
package_build "freebsd" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for FreeBSD (arm64)..."
GOOS=freebsd GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-freebsd-arm64" .
chmod +x "builds/${PROJECT_NAME}-freebsd-arm64"
package_build "freebsd" "arm64" "$PROJECT_NAME" "$VERSION"

echo "All builds completed. ZIP archives are in the 'builds' directory."