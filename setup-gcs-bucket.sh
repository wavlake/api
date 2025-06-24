#!/bin/bash

# GCS Bucket Setup Script for Wavlake Audio Uploads
# This script creates a properly configured bucket for the Nostr track upload API

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🎵 Wavlake GCS Bucket Setup${NC}"
echo "=================================="

# Check if required environment variables are set
if [ -z "$GOOGLE_CLOUD_PROJECT" ]; then
    echo -e "${RED}Error: GOOGLE_CLOUD_PROJECT environment variable must be set${NC}"
    echo "Run: export GOOGLE_CLOUD_PROJECT=your-project-id"
    exit 1
fi

# Prompt for bucket name if not set
if [ -z "$GCS_BUCKET_NAME" ]; then
    echo -e "${YELLOW}Enter bucket name (must be globally unique):${NC}"
    read -p "Bucket name: " GCS_BUCKET_NAME
    
    if [ -z "$GCS_BUCKET_NAME" ]; then
        echo -e "${RED}Error: Bucket name cannot be empty${NC}"
        exit 1
    fi
fi

# Optional: Set region (default to us-central1)
REGION=${GCS_REGION:-"us-central1"}

echo ""
echo -e "${BLUE}Configuration:${NC}"
echo "Project ID: $GOOGLE_CLOUD_PROJECT"
echo "Bucket Name: $GCS_BUCKET_NAME"
echo "Region: $REGION"
echo ""

# Confirm before proceeding
read -p "Continue with bucket creation? (y/N): " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo -e "${YELLOW}Creating GCS bucket...${NC}"

# Create the bucket
gsutil mb -p $GOOGLE_CLOUD_PROJECT -c STANDARD -l $REGION gs://$GCS_BUCKET_NAME

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Bucket created successfully!${NC}"
else
    echo -e "${RED}❌ Failed to create bucket${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}Setting up bucket structure...${NC}"

# Create the directory structure by uploading placeholder files
echo "Setting up tracks/original/ directory..." > /tmp/placeholder.txt
gsutil cp /tmp/placeholder.txt gs://$GCS_BUCKET_NAME/tracks/original/.placeholder
gsutil rm gs://$GCS_BUCKET_NAME/tracks/original/.placeholder

echo "Setting up tracks/compressed/ directory..." > /tmp/placeholder.txt
gsutil cp /tmp/placeholder.txt gs://$GCS_BUCKET_NAME/tracks/compressed/.placeholder
gsutil rm gs://$GCS_BUCKET_NAME/tracks/compressed/.placeholder

# Clean up temp file
rm /tmp/placeholder.txt

echo -e "${GREEN}✅ Directory structure created${NC}"

echo ""
echo -e "${YELLOW}Configuring bucket permissions...${NC}"

# Set bucket to not be publicly readable by default
gsutil iam ch -d allUsers:objectViewer gs://$GCS_BUCKET_NAME 2>/dev/null || true

# Allow public read access to compressed files only (for streaming)
gsutil iam ch allUsers:objectViewer gs://$GCS_BUCKET_NAME/tracks/compressed/*

echo -e "${GREEN}✅ Permissions configured${NC}"

echo ""
echo -e "${YELLOW}Setting up lifecycle policy...${NC}"

# Create lifecycle policy to clean up incomplete uploads
cat > /tmp/lifecycle.json << EOF
{
  "lifecycle": {
    "rule": [
      {
        "condition": {
          "age": 1,
          "isLive": false
        },
        "action": {
          "type": "Delete"
        }
      },
      {
        "condition": {
          "age": 7
        },
        "action": {
          "type": "AbortIncompleteMultipartUpload"
        }
      }
    ]
  }
}
EOF

gsutil lifecycle set /tmp/lifecycle.json gs://$GCS_BUCKET_NAME
rm /tmp/lifecycle.json

echo -e "${GREEN}✅ Lifecycle policy applied${NC}"

echo ""
echo -e "${YELLOW}Setting up CORS policy for browser uploads...${NC}"

# Create CORS policy for direct browser uploads
cat > /tmp/cors.json << EOF
[
  {
    "origin": ["*"],
    "method": ["GET", "PUT", "POST", "HEAD"],
    "responseHeader": ["Content-Type", "ETag"],
    "maxAgeSeconds": 3600
  }
]
EOF

gsutil cors set /tmp/cors.json gs://$GCS_BUCKET_NAME
rm /tmp/cors.json

echo -e "${GREEN}✅ CORS policy configured${NC}"

echo ""
echo -e "${YELLOW}Verifying setup...${NC}"

# Verify bucket exists and show info
gsutil ls -L -b gs://$GCS_BUCKET_NAME

echo ""
echo -e "${GREEN}🎉 GCS Bucket Setup Complete!${NC}"
echo "=================================="
echo ""
echo -e "${BLUE}Bucket Details:${NC}"
echo "• Name: $GCS_BUCKET_NAME"
echo "• URL: https://storage.googleapis.com/$GCS_BUCKET_NAME"
echo "• Region: $REGION"
echo "• Structure:"
echo "  └── tracks/"
echo "      ├── original/     (private - original uploads)"
echo "      └── compressed/   (public - streaming files)"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "1. Export the bucket name:"
echo "   ${YELLOW}export GCS_BUCKET_NAME=$GCS_BUCKET_NAME${NC}"
echo ""
echo "2. Update your API environment variables:"
echo "   ${YELLOW}export GCS_BUCKET_NAME=$GCS_BUCKET_NAME${NC}"
echo ""
echo "3. Deploy the Cloud Function:"
echo "   ${YELLOW}cd cloud-function/ && ./deploy.sh${NC}"
echo ""
echo -e "${GREEN}Your bucket is ready for audio uploads! 🎵${NC}"