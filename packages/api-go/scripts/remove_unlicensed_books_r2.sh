#!/bin/bash
# Deletes the files of the books scripts/migrate_remove_unlicensed_books.sql
# took out of the catalogue. They sit in a public bucket, so until this runs
# anyone with a link can still download them.
#
# Run the migration first, on every database that uses this bucket (local and
# Supabase), so nothing is left pointing at a deleted file.
#
# Requires: awscli (brew install awscli) and .env.local with the R2 settings.
#
# Usage (from packages/api-go):
#   ./scripts/remove_unlicensed_books_r2.sh           # lists what it would delete
#   ./scripts/remove_unlicensed_books_r2.sh --apply   # deletes it

set -euo pipefail

# Only the R2 settings: .env.local is read by godotenv, not a shell, and
# other values in it (the Apple key) don't survive `source`.
env_value() {
  grep -E "^$1=" .env.local | tail -1 | cut -d= -f2- | sed -E 's/^["'\'']//; s/["'\'']$//'
}
R2_ENDPOINT=$(env_value R2_ENDPOINT)
R2_ACCESS_KEY_ID=$(env_value R2_ACCESS_KEY_ID)
R2_SECRET_ACCESS_KEY=$(env_value R2_SECRET_ACCESS_KEY)
R2_BUCKET_NAME=$(env_value R2_BUCKET_NAME)
for v in R2_ENDPOINT R2_ACCESS_KEY_ID R2_SECRET_ACCESS_KEY R2_BUCKET_NAME; do
  if [ -z "${!v}" ]; then
    echo "$v is missing from .env.local" >&2
    exit 1
  fi
done

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY"
S3="aws s3 --endpoint-url $R2_ENDPOINT --region auto"

DRYRUN="--dryrun"
if [ "${1:-}" = "--apply" ]; then
  DRYRUN=""
fi

# Uploaded books keep their files under their id, manuscript and cover alike.
IDS=(
  bk_23c53501 bk_6a77fd96 bk_1fa54678 bk_dc713202
  bk_0055eda3 bk_189ef70f bk_c079ae13 bk_07595f86 bk_b9992a17
  bk_95945c53 bk_a290f1ee bk_704a73bd bk_abd50953 bk_4c70f301
  bk_2d425aaf bk_a866e332 bk_07b1cda2 bk_4bd582a6
  bk_9ad97ab2 bk_48ea0037 bk_b51e2f92 bk_fb9daf65 bk_59b1cd1e
)

# The older seeded books are filed by genre and name instead. Python Notes
# for Professionals, beside them, stays.
SEEDED=(
  "Fantasy/Mythic Fantasy/Art Heist Baby"
  "Non-Fiction/Science/DataMining"
  "Non-Fiction/Science/DataScienceAndPredictiveAnalyt"
  "Non-Fiction/Science/QuantumMechanics"
  "Non-Fiction/Science/RegressionModelingStrategies"
)

ARGS=(--recursive --exclude "*")
for id in "${IDS[@]}"; do
  ARGS+=(--include "manuscripts/$id-*" --include "covers/$id-*")
done
for name in "${SEEDED[@]}"; do
  ARGS+=(--include "manuscripts/$name.*" --include "covers/$name.*")
done

$S3 rm "s3://$R2_BUCKET_NAME/books/" "${ARGS[@]}" $DRYRUN

if [ -n "$DRYRUN" ]; then
  echo ""
  echo "Nothing was deleted. Run again with --apply to delete the files above."
fi
