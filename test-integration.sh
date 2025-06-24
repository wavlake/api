#!/bin/bash

# Integration test script for Wavlake API and Cloud Function

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_URL="https://api-cgi4gylh7q-uc.a.run.app"

echo -e "${BLUE}🧪 Running Wavlake Integration Tests${NC}"
echo "=================================="
echo "API URL: $API_URL"
echo ""

# Test 1: Heartbeat
echo -e "${YELLOW}Test 1: Heartbeat endpoint${NC}"
response=$(curl -s -w "%{http_code}" "$API_URL/heartbeat" -o /tmp/heartbeat_response)
http_code="${response: -3}"

if [ "$http_code" = "200" ]; then
    echo -e "${GREEN}✅ Heartbeat: PASS${NC}"
else
    echo -e "${RED}❌ Heartbeat: FAIL (HTTP $http_code)${NC}"
    cat /tmp/heartbeat_response
fi

echo ""

# Test 2: Webhook endpoint structure
echo -e "${YELLOW}Test 2: Webhook endpoint${NC}"
WEBHOOK_SECRET="ed6c1d3ab34f6af3cc173236e773565545af308a5a681e8c686664506cbfc0e2"
response=$(curl -s -w "%{http_code}" -X POST "$API_URL/v1/tracks/webhook/process" \
  -H "X-Webhook-Secret: $WEBHOOK_SECRET" -o /tmp/webhook_response)
http_code="${response: -3}"

if [ "$http_code" = "400" ]; then
    echo -e "${GREEN}✅ Webhook: PASS (correctly rejects empty payload)${NC}"
else
    echo -e "${RED}❌ Webhook: FAIL (HTTP $http_code)${NC}"
    cat /tmp/webhook_response
fi

echo ""

# Test 3: Protected endpoint without auth
echo -e "${YELLOW}Test 3: Protected endpoint (no auth)${NC}"
response=$(curl -s -w "%{http_code}" -X POST "$API_URL/v1/tracks/nostr" \
  -H "Content-Type: application/json" \
  -d '{"extension": "mp3"}' -o /tmp/noauth_response)
http_code="${response: -3}"

if [ "$http_code" = "401" ]; then
    echo -e "${GREEN}✅ Auth protection: PASS (correctly rejects without auth)${NC}"
else
    echo -e "${RED}❌ Auth protection: FAIL (HTTP $http_code)${NC}"
    cat /tmp/noauth_response
fi

echo ""

# Test 4: Cloud Function connectivity
echo -e "${YELLOW}Test 4: Cloud Function status${NC}"
cf_status=$(gcloud functions describe process-audio-upload --region=us-central1 --format="value(state)" 2>/dev/null || echo "NOT_FOUND")

if [ "$cf_status" = "ACTIVE" ]; then
    echo -e "${GREEN}✅ Cloud Function: ACTIVE${NC}"
else
    echo -e "${RED}❌ Cloud Function: $cf_status${NC}"
fi

echo ""

# Test 5: Environment variables
echo -e "${YELLOW}Test 5: Environment configuration${NC}"
if [ -n "$GCS_BUCKET_NAME" ]; then
    echo -e "${GREEN}✅ GCS_BUCKET_NAME: $GCS_BUCKET_NAME${NC}"
else
    echo -e "${RED}❌ GCS_BUCKET_NAME: Not set${NC}"
fi

if [ -n "$GOOGLE_CLOUD_PROJECT" ]; then
    echo -e "${GREEN}✅ GOOGLE_CLOUD_PROJECT: $GOOGLE_CLOUD_PROJECT${NC}"
else
    echo -e "${RED}❌ GOOGLE_CLOUD_PROJECT: Not set${NC}"
fi

echo ""

# Test 6: GCS bucket accessibility
echo -e "${YELLOW}Test 6: GCS bucket access${NC}"
if gsutil ls "gs://$GCS_BUCKET_NAME/" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ GCS bucket: Accessible${NC}"
    
    # Check if we can write to the bucket
    echo "test" > /tmp/bucket-test.txt
    if gsutil cp /tmp/bucket-test.txt "gs://$GCS_BUCKET_NAME/tracks/original/bucket-test.txt" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ GCS bucket: Writable${NC}"
        gsutil rm "gs://$GCS_BUCKET_NAME/tracks/original/bucket-test.txt" > /dev/null 2>&1
    else
        echo -e "${RED}❌ GCS bucket: Not writable${NC}"
    fi
    rm /tmp/bucket-test.txt 2>/dev/null
else
    echo -e "${RED}❌ GCS bucket: Not accessible${NC}"
fi

echo ""

# Test 7: Cloud Function trigger test
echo -e "${YELLOW}Test 7: Trigger test (optional)${NC}"
read -p "Upload test file to trigger Cloud Function? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    test_file="test-track-$(date +%s).mp3"
    echo "Test content for trigger" > "/tmp/$test_file"
    
    echo "Uploading test file..."
    gsutil cp "/tmp/$test_file" "gs://$GCS_BUCKET_NAME/tracks/original/$test_file"
    
    echo "Waiting 10 seconds for trigger..."
    sleep 10
    
    echo "Recent Cloud Function logs:"
    gcloud functions logs read process-audio-upload --region=us-central1 --limit=3
    
    # Cleanup
    gsutil rm "gs://$GCS_BUCKET_NAME/tracks/original/$test_file" 2>/dev/null || true
    rm "/tmp/$test_file" 2>/dev/null || true
else
    echo "Skipped trigger test"
fi

echo ""
echo -e "${BLUE}🎉 Integration tests complete!${NC}"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "1. Test with a real Nostr client and NIP-98 authentication"
echo "2. Upload actual audio files to test processing"
echo "3. Monitor logs during real usage"
echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo "• API logs: gcloud logs read --project=wavlake-alpha --filter='resource.type=cloud_run_revision'"
echo "• Function logs: gcloud functions logs read process-audio-upload --region=us-central1"
echo "• List files: gsutil ls gs://$GCS_BUCKET_NAME/tracks/"