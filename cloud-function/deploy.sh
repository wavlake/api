#!/bin/bash

# Cloud Function deployment script
# Run this from the cloud-function directory

PROJECT_ID=${GOOGLE_CLOUD_PROJECT}
BUCKET_NAME=${GCS_BUCKET_NAME}
API_BASE_URL=${API_BASE_URL:-"https://your-api-domain.com"}
WEBHOOK_SECRET=${WEBHOOK_SECRET:-""}

if [ -z "$PROJECT_ID" ]; then
    echo "Error: GOOGLE_CLOUD_PROJECT environment variable must be set"
    exit 1
fi

if [ -z "$BUCKET_NAME" ]; then
    echo "Error: GCS_BUCKET_NAME environment variable must be set"
    exit 1
fi

echo "Deploying Cloud Function for audio processing..."
echo "Project: $PROJECT_ID"
echo "Bucket: $BUCKET_NAME"
echo "API URL: $API_BASE_URL"

# Deploy the Cloud Function
gcloud functions deploy process-audio-upload \
    --gen2 \
    --runtime=go122 \
    --region=us-central1 \
    --source=. \
    --entry-point=ProcessAudioUpload \
    --trigger-bucket=$BUCKET_NAME \
    --set-env-vars="API_BASE_URL=$API_BASE_URL,WEBHOOK_SECRET=$WEBHOOK_SECRET" \
    --memory=512MB \
    --timeout=540s \
    --max-instances=10 \
    --project=$PROJECT_ID

if [ $? -eq 0 ]; then
    echo "Cloud Function deployed successfully!"
    echo ""
    echo "The function will now automatically trigger when files are uploaded to:"
    echo "gs://$BUCKET_NAME/tracks/original/"
    echo ""
    echo "Make sure your API webhook endpoint is accessible at:"
    echo "$API_BASE_URL/v1/tracks/webhook/process"
else
    echo "Deployment failed!"
    exit 1
fi