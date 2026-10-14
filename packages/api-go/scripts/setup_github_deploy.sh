#!/usr/bin/env bash
# One-time Google Cloud setup for .github/workflows/deploy-api.yml.
#
# GitHub Actions signs in with Workload Identity Federation: GitHub's OIDC token
# is exchanged for a short-lived Google one, so no service-account key is ever
# stored in GitHub. Google only honours tokens from Mavzz/readpanda's main
# branch.
#
# Re-runnable: anything that already exists is left as it is. Run it as a
# project owner:
#   bash packages/api-go/scripts/setup_github_deploy.sh
set -euo pipefail

PROJECT_ID=reanpanda-d3beb
REGION=asia-south1
REPO=Mavzz/readpanda
POOL=github
PROVIDER=readpanda-repo
SA_NAME=github-deployer
SOURCE_BUCKET="run-sources-${PROJECT_ID}-${REGION}"
AR_REPO=cloud-run-source-deploy

PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
SA="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
# The service runs as, and Cloud Build builds as, the default compute account.
RUNTIME_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

echo "== APIs"
# sts exchanges the GitHub token; iamcredentials mints the service account's.
gcloud services enable sts.googleapis.com iamcredentials.googleapis.com \
  run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  --project "$PROJECT_ID"

echo "== Workload identity pool and provider"
gcloud iam workload-identity-pools describe "$POOL" \
  --project "$PROJECT_ID" --location global >/dev/null 2>&1 ||
  gcloud iam workload-identity-pools create "$POOL" \
    --project "$PROJECT_ID" --location global \
    --display-name "GitHub Actions"

gcloud iam workload-identity-pools providers describe "$PROVIDER" \
  --project "$PROJECT_ID" --location global --workload-identity-pool "$POOL" >/dev/null 2>&1 ||
  gcloud iam workload-identity-pools providers create-oidc "$PROVIDER" \
    --project "$PROJECT_ID" --location global --workload-identity-pool "$POOL" \
    --display-name "Mavzz/readpanda" \
    --issuer-uri "https://token.actions.githubusercontent.com" \
    --attribute-mapping "google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
    --attribute-condition "assertion.repository == '${REPO}' && assertion.ref == 'refs/heads/main'"

echo "== Deployer service account"
gcloud iam service-accounts describe "$SA" --project "$PROJECT_ID" >/dev/null 2>&1 ||
  gcloud iam service-accounts create "$SA_NAME" \
    --project "$PROJECT_ID" \
    --display-name "GitHub Actions deployer (deploy-api.yml)"

# Only this repo's workflows (and, by the provider's condition, only on main)
# may act as the deployer.
gcloud iam service-accounts add-iam-policy-binding "$SA" \
  --project "$PROJECT_ID" \
  --role roles/iam.workloadIdentityUser \
  --member "principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/attribute.repository/${REPO}" \
  --condition None >/dev/null

echo "== What the deployer may do"
# Deploy revisions (not change who can invoke the service: that's run.admin),
# start the source build, and read its logs while it streams.
for role in roles/run.developer roles/cloudbuild.builds.editor \
  roles/logging.viewer roles/serviceusage.serviceUsageConsumer; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" \
    --member "serviceAccount:${SA}" --role "$role" --condition None >/dev/null
done

# Deploying a revision that runs as the compute account, and a build that
# builds as it, means acting as it — on that account only.
gcloud iam service-accounts add-iam-policy-binding "$RUNTIME_SA" \
  --project "$PROJECT_ID" \
  --member "serviceAccount:${SA}" --role roles/iam.serviceAccountUser \
  --condition None >/dev/null

# `--source .` uploads the code to this bucket and checks the image repo exists.
gcloud storage buckets add-iam-policy-binding "gs://${SOURCE_BUCKET}" \
  --member "serviceAccount:${SA}" --role roles/storage.admin >/dev/null
gcloud artifacts repositories add-iam-policy-binding "$AR_REPO" \
  --project "$PROJECT_ID" --location "$REGION" \
  --member "serviceAccount:${SA}" --role roles/artifactregistry.reader >/dev/null

echo
echo "Done. workload_identity_provider for deploy-api.yml:"
echo "  projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL}/providers/${PROVIDER}"
echo "service_account:"
echo "  ${SA}"
echo
echo "Then: gh workflow enable deploy-api.yml && gh workflow run deploy-api.yml --ref main"
