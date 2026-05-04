#!/usr/bin/env sh
set -eu

install_dir="${TOKENMETER_INSTALL_DIR:-$HOME/.local/bin}"
bin_path="$install_dir/tkm"
tmp_path="$install_dir/tkm.tmp"
url="https://github.com/otechmista/token-meter/releases/latest/download/tkm-linux-amd64"
is_update=0

if [ -f "$bin_path" ]; then
  is_update=1
fi

echo "TokenMeter install/update"
echo "Target: $bin_path"

mkdir -p "$install_dir"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_path"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_path" "$url"
else
  echo "error: install curl or wget first" >&2
  exit 1
fi

mv -f "$tmp_path" "$bin_path"
chmod +x "$bin_path"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    profile="$HOME/.profile"
    line="export PATH=\"$install_dir:\$PATH\""
    if [ ! -f "$profile" ] || ! grep -F "$line" "$profile" >/dev/null 2>&1; then
      printf '\n%s\n' "$line" >> "$profile"
    fi
    echo "Added TokenMeter to PATH in $profile."
    echo "Open a new terminal or run: . $profile"
    ;;
esac

if [ "$is_update" -eq 1 ]; then
  echo "Updated existing TokenMeter."
else
  echo "Installed TokenMeter."
fi

"$bin_path" --top 0 "$PWD"
echo ""
echo "Run: tkm ."
