#!/bin/bash
# Uploads sample book covers and manuscripts to R2 for testing.
# Requires: awscli (brew install awscli)
#
# Usage: ./scripts/seed_r2.sh

set -euo pipefail

# Load env vars from .env.local
source .env.local

# Configure the S3 endpoint alias
export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
ENDPOINT="$R2_ENDPOINT"
BUCKET="$R2_BUCKET_NAME"

S3="aws s3 --endpoint-url $ENDPOINT --region auto"

# Create a temp directory for sample files
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# --- Sample book definitions: genre/subgenre/title ---
declare -a BOOKS=(
  "Fiction/Fantasy/The Dragon Chronicles"
  "Fiction/Fantasy/Elven Kingdoms"
  "Fiction/Science Fiction/Mars Colony"
  "Fiction/Science Fiction/The Quantum Gate"
  "Fiction/Romance/Letters in the Rain"
  "Fiction/Mystery/The Silent Witness"
  "Fiction/Thriller/Code Red"
  "Non-Fiction/Biography/Life of Tesla"
  "Non-Fiction/Self-Help/Atomic Focus"
  "Non-Fiction/History/The Silk Road"
)

echo "Generating sample files..."

for entry in "${BOOKS[@]}"; do
  genre=$(echo "$entry" | cut -d/ -f1)
  subgenre=$(echo "$entry" | cut -d/ -f2)
  title=$(echo "$entry" | cut -d/ -f3)

  # Create directories
  mkdir -p "$TMP/books/covers/$genre/$subgenre"
  mkdir -p "$TMP/books/manuscripts/$genre/$subgenre"

  # Generate a simple SVG cover (valid image file)
  COLOR=$(printf '%06X' $((RANDOM * RANDOM % 16777215)))
  cat > "$TMP/books/covers/$genre/$subgenre/$title.svg" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="400" height="600">
  <rect width="400" height="600" fill="#$COLOR"/>
  <text x="200" y="250" text-anchor="middle" font-size="28" fill="white" font-family="sans-serif">$title</text>
  <text x="200" y="300" text-anchor="middle" font-size="16" fill="white" font-family="sans-serif">$genre / $subgenre</text>
</svg>
EOF

  # Generate a simple text manuscript
  cat > "$TMP/books/manuscripts/$genre/$subgenre/$title.txt" <<EOF
$title
Genre: $genre | Subgenre: $subgenre

Chapter 1

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo.

Chapter 2

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore
eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
sunt in culpa qui officia deserunt mollit anim id est laborum.
EOF

done

echo "Uploading to R2 bucket: $BUCKET ..."

# Upload covers
$S3 cp "$TMP/books/covers/" "s3://$BUCKET/books/covers/" \
  --recursive --content-type "image/svg+xml" --quiet

# Upload manuscripts
$S3 cp "$TMP/books/manuscripts/" "s3://$BUCKET/books/manuscripts/" \
  --recursive --content-type "text/plain" --quiet

echo ""
echo "Done! Uploaded ${#BOOKS[@]} books:"
for entry in "${BOOKS[@]}"; do
  echo "  - $entry"
done
echo ""
echo "Now call POST /api/v1/books/seed to populate the database."
