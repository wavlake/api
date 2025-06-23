#!/bin/bash

# Cloud Build Trigger Setup Script for Wavlake API

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Setting up Cloud Build trigger for automatic deployments...${NC}"

# Get project ID
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}Error: No project configured. Run 'gcloud config set project YOUR_PROJECT_ID'${NC}"
    exit 1
fi

echo -e "${YELLOW}Using project: $PROJECT_ID${NC}"

# Enable required APIs
echo -e "${YELLOW}Enabling required APIs...${NC}"
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable secretmanager.googleapis.com

# Create service account if it doesn't exist
SERVICE_ACCOUNT="api-service@$PROJECT_ID.iam.gserviceaccount.com"
if ! gcloud iam service-accounts describe $SERVICE_ACCOUNT &>/dev/null; then
    echo -e "${YELLOW}Creating service account...${NC}"
    gcloud iam service-accounts create api-service \
        --display-name="API Service Account" \
        --description="Service account for Wavlake API Cloud Run service"
else
    echo -e "${GREEN}Service account already exists${NC}"
fi

# Grant necessary permissions
echo -e "${YELLOW}Granting IAM permissions...${NC}"
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/datastore.user"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/logging.logWriter"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/monitoring.metricWriter"

# Grant Cloud Build service account permissions to deploy to Cloud Run
CLOUD_BUILD_SA="$PROJECT_ID@cloudbuild.gserviceaccount.com"
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/run.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/iam.serviceAccountUser"

echo -e "${GREEN}✅ Cloud Build setup completed!${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Connect your GitHub repository in Cloud Console:"
echo "   https://console.cloud.google.com/cloud-build/triggers"
echo ""
echo "2. Create a trigger with these settings:"
echo "   - Repository: wavlake/api"
echo "   - Branch: ^main$"
echo "   - Configuration: Cloud Build configuration file"
echo "   - File location: cloudbuild.yaml"
echo ""
echo "3. Push to main branch to trigger your first deployment!"
echo ""
echo -e "${GREEN}Repository will auto-deploy on every push to main branch.${NC}"