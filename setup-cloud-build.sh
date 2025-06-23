#!/bin/bash

# Cloud Build Setup Script for Wavlake API
# This script automates the initial setup for Cloud Build and Cloud Run

set -e

echo "Cloud Build Setup for Wavlake API"
echo "================================="

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo "Error: gcloud CLI is not installed. Please install it first."
    echo "Visit: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Get or set project ID
if [ -z "$PROJECT_ID" ]; then
    echo -n "Enter your Google Cloud Project ID: "
    read PROJECT_ID
fi

echo "Using project: $PROJECT_ID"

# Set the project
gcloud config set project $PROJECT_ID

# Enable required APIs
echo ""
echo "Enabling required APIs..."
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable secretmanager.googleapis.com

# Create service account
echo ""
echo "Creating service account..."
if ! gcloud iam service-accounts describe api-service@$PROJECT_ID.iam.gserviceaccount.com &> /dev/null; then
    gcloud iam service-accounts create api-service \
        --display-name="API Service Account"
    echo "Service account created."
else
    echo "Service account already exists."
fi

# Grant permissions to service account
echo ""
echo "Granting permissions to service account..."
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:api-service@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/run.invoker"

# Get Cloud Build service account
CLOUD_BUILD_SA=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')@cloudbuild.gserviceaccount.com

# Grant Cloud Build permissions
echo ""
echo "Granting permissions to Cloud Build..."
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/run.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/iam.serviceAccountUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/storage.admin"

echo ""
echo "Setup completed successfully!"
echo ""
echo "Next steps:"
echo "1. Connect your GitHub repository in the Cloud Console:"
echo "   https://console.cloud.google.com/cloud-build/triggers"
echo ""
echo "2. Create a trigger using the Cloud Console or run:"
echo "   gcloud builds triggers create github \\"
echo "       --repo-name=api \\"
echo "       --repo-owner=wavlake \\"
echo "       --branch-pattern=\"^main$\" \\"
echo "       --build-config=cloudbuild-trigger.yaml \\"
echo "       --name=api-main-deploy"
echo ""
echo "3. Review the CLOUD_BUILD_SETUP.md file for detailed instructions"
echo ""
echo "Project ID: $PROJECT_ID"
echo "Service Account: api-service@$PROJECT_ID.iam.gserviceaccount.com"
echo "Cloud Build SA: $CLOUD_BUILD_SA"