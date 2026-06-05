#!/bin/bash

# RSA Key Generation Script for Vega Services
# Usage: ./generate-keys.sh [output_dir]
# Default output directory: current directory
# Skips key generation if any target *.pem already exists.
# Also derives wga-sandbox-ontology/state.json from the public key (skipped if exists).
# To regenerate, delete the existing files first.

set -e

OUTPUT_DIR="${1:-.}"

echo "=== RSA Key Generation Script ==="
echo "Output directory: $OUTPUT_DIR"
echo ""

DC_PRIV="$OUTPUT_DIR/data-connection/private_key.pem"
DC_PUB="$OUTPUT_DIR/data-connection/public_key.pem"
GW_PRIV="$OUTPUT_DIR/vega-gateway-pro/private_key.pem"
STATE_JSON="$OUTPUT_DIR/wga-sandbox-ontology/state.json"

EXISTING=()
[ -f "$DC_PRIV" ] && EXISTING+=("$DC_PRIV")
[ -f "$DC_PUB" ]  && EXISTING+=("$DC_PUB")
[ -f "$GW_PRIV" ] && EXISTING+=("$GW_PRIV")

if [ ${#EXISTING[@]} -gt 0 ]; then
  echo "Existing key files detected, skipping key generation:"
  for f in "${EXISTING[@]}"; do
    echo "  - $f"
  done
  echo ""
  echo "Delete these files first if you want to regenerate."
else
  mkdir -p "$OUTPUT_DIR/data-connection"
  mkdir -p "$OUTPUT_DIR/vega-gateway-pro"

  echo "Generating RSA private key..."
  openssl genrsa -out "$DC_PRIV" 2048 2>/dev/null
  echo "  Created: $DC_PRIV"

  echo "Generating RSA public key..."
  openssl rsa -in "$DC_PRIV" -pubout -out "$DC_PUB" 2>/dev/null
  echo "  Created: $DC_PUB"

  echo "Copying private key to vega-gateway-pro..."
  cp "$DC_PRIV" "$GW_PRIV"
  echo "  Created: $GW_PRIV"

  echo "Setting file permissions ..."
  chmod 644 "$DC_PRIV"
  chmod 644 "$DC_PUB"
  chmod 644 "$GW_PRIV"

  echo ""
  echo "=== RSA Key Generation Complete ==="
fi

echo ""
if [ -f "$STATE_JSON" ]; then
  echo "Existing state.json detected, skipping: $STATE_JSON"
  echo "Delete it first if you want to regenerate."
elif [ -f "$DC_PUB" ]; then
  echo "Generating state.json from public key..."
  PEM_ESCAPED=$(awk 'BEGIN { ORS="" } { sub(/\r$/, ""); if (NR>1) printf "\\n"; printf "%s", $0 }' "$DC_PUB")
  mkdir -p "$(dirname "$STATE_JSON")"
  cat > "$STATE_JSON" << EOF
{
  "publicKey": "${PEM_ESCAPED}"
}
EOF
  echo "  Created: $STATE_JSON"
else
  echo "Public key not found ($DC_PUB), skipping state.json generation."
fi

echo ""
echo "IMPORTANT: Do NOT commit generated key/state files to version control!"
