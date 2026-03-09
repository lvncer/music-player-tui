#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

FW="MediaRemoteAdapter.framework"

if [[ ! -d "$FW" ]]; then
  echo "Missing $FW in $DIR" >&2
  exit 1
fi

# electron-builder が .framework 内の symlink をうまく扱えず壊すことがあるので、
# Versions/Current を実体ディレクトリとして用意しておく（Aを複製）。

if [[ -L "$FW/Versions/Current" ]]; then
  echo "Converting $FW/Versions/Current from symlink to directory..."
  TARGET="$(readlink "$FW/Versions/Current")"
  rm "$FW/Versions/Current"
  mkdir -p "$FW/Versions/Current"
  # Copy contents of Versions/$TARGET into Versions/Current
  cp -R "$FW/Versions/$TARGET/"* "$FW/Versions/Current/"
fi

# Ensure targets exist even if root symlinks get dropped during packaging.
mkdir -p "$FW/Versions/Current/Resources"

if [[ ! -f "$FW/Versions/Current/MediaRemoteAdapter" ]]; then
  if [[ -f "$FW/Versions/A/MediaRemoteAdapter" ]]; then
    cp "$FW/Versions/A/MediaRemoteAdapter" "$FW/Versions/Current/MediaRemoteAdapter"
  fi
fi

if [[ ! -f "$FW/Versions/Current/Resources/Info.plist" ]]; then
  if [[ -f "$FW/Versions/A/Resources/Info.plist" ]]; then
    cp "$FW/Versions/A/Resources/Info.plist" "$FW/Versions/Current/Resources/Info.plist"
  fi
fi

echo "OK: normalized $FW for electron-builder."

