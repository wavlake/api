#!/bin/bash

# Cloud Run deployment script for Wavlake API

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 Deploying Wavlake API to Cloud Run${NC}"
echo "======================================"

# Check required environment variables
if [ -z "$GOOGLE_CLOUD_PROJECT" ]; then
    echo -e "${RED}Error: GOOGLE_CLOUD_PROJECT environment variable must be set${NC}"
    exit 1
fi

if [ -z "$GCS_BUCKET_NAME" ]; then
    echo -e "${YELLOW}GCS_BUCKET_NAME not set. Attempting to detect...${NC}"
    
    # Try to get the first bucket that looks like it's for audio uploads
    GCS_BUCKET_NAME=$(gsutil ls | grep -E "(audio|wavlake|track)" | head -1 | sed 's|gs://||' | sed 's|/||')
    
    if [ -z "$GCS_BUCKET_NAME" ]; then
        echo -e "${RED}Error: Could not detect GCS bucket. Please set GCS_BUCKET_NAME${NC}"
        echo "Available buckets:"
        gsutil ls
        echo ""
        echo "Set with: export GCS_BUCKET_NAME=your-bucket-name"
        exit 1
    else
        echo -e "${GREEN}Detected bucket: $GCS_BUCKET_NAME${NC}"
    fi
fi

# Optional variables with defaults
REGION=${REGION:-"us-central1"}
SERVICE_NAME=${SERVICE_NAME:-"api"}
WEBHOOK_SECRET=${WEBHOOK_SECRET:-$(openssl rand -hex 32)}

echo -e "${BLUE}Configuration:${NC}"
echo "Project: $GOOGLE_CLOUD_PROJECT"
echo "Region: $REGION"
echo "Service: $SERVICE_NAME"
echo "Bucket: $GCS_BUCKET_NAME"
echo "Webhook Secret: ${WEBHOOK_SECRET:0:8}..."
echo ""

# Build and deploy
echo -e "${YELLOW}Building and deploying to Cloud Run...${NC}"

gcloud run deploy $SERVICE_NAME \
    --source=. \
    --region=$REGION \
    --allow-unauthenticated \
    --set-env-vars="GOOGLE_CLOUD_PROJECT=$GOOGLE_CLOUD_PROJECT,GCS_BUCKET_NAME=$GCS_BUCKET_NAME,WEBHOOK_SECRET=$WEBHOOK_SECRET" \
    --memory=1Gi \
    --cpu=1 \
    --timeout=300 \
    --max-instances=10 \
    --project=$GOOGLE_CLOUD_PROJECT

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Deployment successful!${NC}"
    
    # Get the service URL
    SERVICE_URL=$(gcloud run services describe $SERVICE_NAME --region=$REGION --project=$GOOGLE_CLOUD_PROJECT --format="value(status.url)")
    
    echo ""
    echo -e "${BLUE}Service Details:${NC}"
    echo "URL: $SERVICE_URL"
    echo "Heartbeat: $SERVICE_URL/heartbeat"
    echo ""
    
    # Test the service
    echo -e "${YELLOW}Testing deployment...${NC}"
    curl -s $SERVICE_URL/heartbeat && echo "" && echo -e "${GREEN}✅ Service is responding!${NC}" || echo -e "${RED}❌ Service not responding${NC}"
    
    echo ""
    echo -e "${BLUE}Next Steps:${NC}"
    echo "1. Update your Cloud Function with:"
    echo "   ${YELLOW}export API_BASE_URL=$SERVICE_URL${NC}"
    echo ""
    echo "2. Deploy/update the Cloud Function:"
    echo "   ${YELLOW}cd cloud-function/ && ./deploy.sh${NC}"
    echo ""
    echo "3. Test the full flow:"
    echo "   ${YELLOW}curl $SERVICE_URL/v1/tracks/webhook/process${NC}"
    
else
    echo -e "${RED}❌ Deployment failed!${NC}"
    echo ""
    echo "Check logs:"
    echo "gcloud logs read --project=$GOOGLE_CLOUD_PROJECT --limit=50 --filter=\"resource.type=cloud_run_revision AND resource.labels.service_name=$SERVICE_NAME\""
    exit 1
fi