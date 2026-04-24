#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <version> [package-name] [repo-url]"
  echo "Example: $0 1.0.0 @thevandieg/postx https://github.com/thevandieg/postcli.git"
  exit 1
fi

VERSION="$1"
PACKAGE_NAME="${2:-@thevandieg/postx}"
REPO_URL="${3:-https://github.com/thevandieg/postcli.git}"
DESCRIPTION="Minimal CLI to compose, schedule, and publish social posts (X only in v${VERSION})"
AUTHOR="thevandieg"
LICENSE_NAME="MIT"

MAIN_PACKAGE_DIR="npm-package"
PLATFORM_PACKAGES_DIR="platform-packages"
DIST_DIR="dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Missing $DIST_DIR directory. Put release archives in $DIST_DIR first."
  exit 1
fi

rm -rf "$MAIN_PACKAGE_DIR" "$PLATFORM_PACKAGES_DIR"
mkdir -p "$MAIN_PACKAGE_DIR/bin" "$PLATFORM_PACKAGES_DIR"

declare -A OS_MAP=(
  ["darwin-x64"]="darwin"
  ["darwin-arm64"]="darwin"
  ["linux-x64"]="linux"
  ["linux-arm64"]="linux"
  ["win32-x64"]="win32"
  ["win32-arm64"]="win32"
)

declare -A CPU_MAP=(
  ["darwin-x64"]="x64"
  ["darwin-arm64"]="arm64"
  ["linux-x64"]="x64"
  ["linux-arm64"]="arm64"
  ["win32-x64"]="x64"
  ["win32-arm64"]="arm64"
)

# Supports both simple names (postx-linux-amd64.tar.gz) and goreleaser-style
# names (postx_1.0.0_Linux_x86_64.tar.gz).
get_platform_keys_for_archive() {
  local file_name="$1"

  case "$file_name" in
    postx-linux-amd64|postx_"${VERSION}"_Linux_x86_64)
      echo "linux-x64"
      ;;
    postx-linux-arm64|postx_"${VERSION}"_Linux_arm64)
      echo "linux-arm64"
      ;;
    postx-darwin-amd64|postx_"${VERSION}"_Darwin_x86_64)
      echo "darwin-x64"
      ;;
    postx-darwin-arm64|postx_"${VERSION}"_Darwin_arm64)
      echo "darwin-arm64"
      ;;
    postx_"${VERSION}"_Darwin_all)
      echo "darwin-x64,darwin-arm64"
      ;;
    postx-windows-amd64|postx_"${VERSION}"_Windows_x86_64)
      echo "win32-x64"
      ;;
    postx-windows-arm64|postx_"${VERSION}"_Windows_arm64)
      echo "win32-arm64"
      ;;
    *)
      echo ""
      ;;
  esac
}

OPTIONAL_DEPS=""

for archive in "$DIST_DIR"/*.tar.gz "$DIST_DIR"/*.zip; do
  [ -f "$archive" ] || continue

  archive_name="$(basename "$archive")"
  archive_name="${archive_name%.tar.gz}"
  archive_name="${archive_name%.zip}"

  platform_keys="$(get_platform_keys_for_archive "$archive_name")"
  if [ -z "$platform_keys" ]; then
    echo "Skipping unknown archive naming: $archive_name"
    continue
  fi

  echo "Processing $archive for platforms: $platform_keys"
  IFS=',' read -ra PLATFORM_ARRAY <<< "$platform_keys"

  for platform_key in "${PLATFORM_ARRAY[@]}"; do
    platform_key="$(echo "$platform_key" | xargs)"
    platform_package_dir="$PLATFORM_PACKAGES_DIR/postx-$platform_key"
    mkdir -p "$platform_package_dir/bin"

    echo "  Creating package for platform: $platform_key"

    if [[ "$archive" == *.tar.gz ]]; then
      tar -xzf "$archive" -C "$platform_package_dir/bin"
    else
      unzip -j "$archive" -d "$platform_package_dir/bin"
    fi

    for doc_file in README.md README README.txt LICENSE LICENSE.md LICENSE.txt; do
      if [ -f "$platform_package_dir/bin/$doc_file" ]; then
        mv "$platform_package_dir/bin/$doc_file" "$platform_package_dir/"
      fi
    done

    binary_name="postx"
    if [[ "$platform_key" == win32-* ]]; then
      binary_name="postx.exe"
    fi

    # Normalize extracted binary name if archive has postx-* naming.
    if [ ! -f "$platform_package_dir/bin/$binary_name" ]; then
      found_bin=""
      for candidate in "$platform_package_dir/bin"/postx "$platform_package_dir/bin"/postx-* "$platform_package_dir/bin"/postx_*.exe "$platform_package_dir/bin"/postx.exe "$platform_package_dir/bin"/postx-*.exe; do
        if [ -f "$candidate" ]; then
          found_bin="$candidate"
          break
        fi
      done
      if [ -n "$found_bin" ]; then
        mv "$found_bin" "$platform_package_dir/bin/$binary_name"
      fi
    fi

    chmod +x "$platform_package_dir/bin/"* || true

    os_value="${OS_MAP[$platform_key]}"
    cpu_value="${CPU_MAP[$platform_key]}"

    files_array='["bin/"]'
    for doc_file in README.md README README.txt LICENSE LICENSE.md LICENSE.txt; do
      if [ -f "$platform_package_dir/$doc_file" ]; then
        files_array="${files_array%]}, \"$doc_file\"]"
      fi
    done

    cat > "$platform_package_dir/package.json" << EOF
{
  "name": "$PACKAGE_NAME-$platform_key",
  "version": "$VERSION",
  "description": "Platform-specific binary for $PACKAGE_NAME ($platform_key)",
  "os": ["$os_value"],
  "cpu": ["$cpu_value"],
  "bin": {
    "postx": "bin/$binary_name"
  },
  "files": $files_array,
  "repository": {
    "type": "git",
    "url": "$REPO_URL"
  },
  "author": "$AUTHOR",
  "license": "$LICENSE_NAME"
}
EOF

    if [ -n "$OPTIONAL_DEPS" ]; then
      OPTIONAL_DEPS="$OPTIONAL_DEPS,"
    fi
    OPTIONAL_DEPS="$OPTIONAL_DEPS\"$PACKAGE_NAME-$platform_key\": \"$VERSION\""
  done
done

if [ -z "$OPTIONAL_DEPS" ]; then
  echo "No matching archives found in $DIST_DIR. Nothing to package."
  exit 1
fi

cat > "$MAIN_PACKAGE_DIR/bin/postx" << EOF
#!/usr/bin/env node

const { execFileSync } = require("node:child_process");

const packageName = "$PACKAGE_NAME";
const platformPackages = {
  "darwin-x64": \`\${packageName}-darwin-x64\`,
  "darwin-arm64": \`\${packageName}-darwin-arm64\`,
  "linux-x64": \`\${packageName}-linux-x64\`,
  "linux-arm64": \`\${packageName}-linux-arm64\`,
  "win32-x64": \`\${packageName}-win32-x64\`,
  "win32-arm64": \`\${packageName}-win32-arm64\`
};

function getBinaryPath() {
  const platformKey = \`\${process.platform}-\${process.arch}\`;
  const platformPackageName = platformPackages[platformKey];

  if (!platformPackageName) {
    console.error(\`Platform \${platformKey} is not supported.\`);
    process.exit(1);
  }

  const binaryName = process.platform === "win32" ? "postx.exe" : "postx";
  try {
    return require.resolve(\`\${platformPackageName}/bin/\${binaryName}\`);
  } catch (err) {
    console.error(\`Missing platform package: \${platformPackageName}\`);
    process.exit(1);
  }
}

try {
  const binaryPath = getBinaryPath();
  execFileSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });
} catch (error) {
  console.error("Failed to execute postx:", error.message);
  process.exit(1);
}
EOF

chmod +x "$MAIN_PACKAGE_DIR/bin/postx"

cat > "$MAIN_PACKAGE_DIR/index.js" << EOF
const { execFileSync } = require("node:child_process");

const packageName = "$PACKAGE_NAME";
const platformPackages = {
  "darwin-x64": \`\${packageName}-darwin-x64\`,
  "darwin-arm64": \`\${packageName}-darwin-arm64\`,
  "linux-x64": \`\${packageName}-linux-x64\`,
  "linux-arm64": \`\${packageName}-linux-arm64\`,
  "win32-x64": \`\${packageName}-win32-x64\`,
  "win32-arm64": \`\${packageName}-win32-arm64\`
};

function getBinaryPath() {
  const platformKey = \`\${process.platform}-\${process.arch}\`;
  const platformPackageName = platformPackages[platformKey];
  if (!platformPackageName) {
    throw new Error(\`Platform \${platformKey} is not supported.\`);
  }
  const binaryName = process.platform === "win32" ? "postx.exe" : "postx";
  return require.resolve(\`\${platformPackageName}/bin/\${binaryName}\`);
}

module.exports = {
  getBinaryPath,
  run(...args) {
    const binaryPath = getBinaryPath();
    return execFileSync(binaryPath, args, { stdio: "inherit" });
  }
};
EOF

cat > "$MAIN_PACKAGE_DIR/package.json" << EOF
{
  "name": "$PACKAGE_NAME",
  "version": "$VERSION",
  "description": "$DESCRIPTION",
  "main": "index.js",
  "bin": {
    "postx": "bin/postx"
  },
  "optionalDependencies": {
    $OPTIONAL_DEPS
  },
  "keywords": ["postx", "cli", "go", "social", "scheduler"],
  "author": "$AUTHOR",
  "license": "$LICENSE_NAME",
  "repository": {
    "type": "git",
    "url": "$REPO_URL"
  },
  "homepage": "${REPO_URL%.git}",
  "engines": {
    "node": ">=18.0.0"
  },
  "files": [
    "bin/",
    "index.js",
    "README.md"
  ]
}
EOF

first_platform_dir=""
for d in "$PLATFORM_PACKAGES_DIR"/*; do
  if [ -d "$d" ]; then
    first_platform_dir="$d"
    break
  fi
done
if [ -n "$first_platform_dir" ] && [ -f "$first_platform_dir/README.md" ]; then
  cp "$first_platform_dir/README.md" "$MAIN_PACKAGE_DIR/"
elif [ -f "README.md" ]; then
  cp "README.md" "$MAIN_PACKAGE_DIR/"
fi

echo
echo "Done."
echo "Main package: $MAIN_PACKAGE_DIR"
echo "Platform packages: $PLATFORM_PACKAGES_DIR"
echo
echo "Next:"
echo "  1) cd $MAIN_PACKAGE_DIR && npm publish --access public"
echo "  2) for d in ../$PLATFORM_PACKAGES_DIR/*; do (cd \"\$d\" && npm publish --access public); done"
