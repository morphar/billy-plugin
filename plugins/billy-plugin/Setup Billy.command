#!/bin/sh
set -eu

plugin_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
"$plugin_dir/scripts/setup-macos"

echo
echo "Billy credential setup is complete. Press Return to close this window."
read -r _
