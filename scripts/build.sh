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

  echo "Moving executable (from builds/${source_artifact_basename}) to staging..."
  cp "builds/${source_artifact_basename}" "${package_content_root}/${executable_name_in_package}"

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
  
  (cd "${temp_staging_dir}" && zip -qr "../${zip_filename_only}" "${project_name_global}")
  
  # Clean up original artifacts after packaging
  rm -f "builds/${source_artifact_basename}"
  
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
echo "Building for macOS (amd64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-macos-amd64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-macos-amd64"
package_build "macos" "amd64" "$PROJECT_NAME" "$VERSION"

echo "Building for macOS (arm64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="${LD_FLAGS}" -o "builds/${PROJECT_NAME}-macos-arm64" "${MAIN_PACKAGE_PATH}"
chmod +x "builds/${PROJECT_NAME}-macos-arm64"
package_build "macos" "arm64" "$PROJECT_NAME" "$VERSION"

echo "All builds completed. ZIP archives are in the 'builds' directory." 