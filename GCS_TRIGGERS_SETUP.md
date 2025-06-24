# GCS Event Triggers Setup

This guide shows how to set up automatic processing when files are uploaded to Google Cloud Storage, replacing the 30-second delay with immediate event-driven processing using Cloud Functions.

## Setup Steps

1. **Navigate to Cloud Function directory**:
   ```bash
   cd cloud-function/
   ```

2. **Set environment variables**:
   ```bash
   export GOOGLE_CLOUD_PROJECT=your-project-id
   export GCS_BUCKET_NAME=your-bucket-name
   export API_BASE_URL=https://your-api-domain.com
   export WEBHOOK_SECRET=your-secret-key  # Optional but recommended
   ```

3. **Deploy the Cloud Function**:
   ```bash
   ./deploy.sh
   ```

## How it Works

1. **File Upload**: Client uploads file to `gs://bucket/tracks/original/uuid.ext`
2. **GCS Trigger**: Cloud Storage automatically triggers the Cloud Function
3. **Function Logic**: 
   - Validates file path (`tracks/original/` only)
   - Extracts track ID from filename
   - Calls API webhook with `status: "uploaded"`
4. **API Processing**: API receives webhook and starts async processing

## Advantages

- ✅ Immediate processing (< 1 second delay)
- ✅ Serverless and auto-scaling
- ✅ Built-in retry logic
- ✅ Simple setup and monitoring
- ✅ Cost-effective (pay per invocation)

## Environment Variables for Cloud Function

```bash
API_BASE_URL=https://your-api.com      # Required: Your API base URL
WEBHOOK_SECRET=your-secret             # Optional: Webhook authentication
```

## Why This Approach

Cloud Functions with GCS triggers is the ideal solution because:

1. **Simplicity**: Single file deployment with automatic GCS triggers
2. **Reliability**: Built-in retry and error handling
3. **Cost**: Only pay when files are uploaded
4. **Monitoring**: Integrated with Cloud Logging and Monitoring
5. **Maintenance**: Minimal operational overhead

## API Changes Required

The API webhook handler has been updated to support the new event flow:

### New Webhook Payload
```json
{
  "track_id": "uuid",
  "status": "uploaded",
  "source": "gcs_trigger"
}
```

### Enhanced Security
```bash
# Set webhook secret for authentication
export WEBHOOK_SECRET=your-secret-key
```

The API now validates the `X-Webhook-Secret` header if `WEBHOOK_SECRET` is set.

## Deployment Checklist

- [ ] Set up GCS bucket with proper permissions
- [ ] Configure API environment variables (including WEBHOOK_SECRET)
- [ ] Deploy Cloud Function with ./deploy.sh
- [ ] Test with file upload
- [ ] Monitor logs for proper event flow
- [ ] Set up alerting for failed processing

## Testing the Setup

1. **Upload a test file**:
   ```bash
   # Create test track via API
   curl -X POST https://your-api.com/v1/tracks/nostr \
     -H "Authorization: Nostr <base64-event>" \
     -d '{"extension": "mp3"}'
   
   # Upload file using presigned URL
   curl -X PUT "presigned-url" --data-binary @test-audio.mp3
   ```

2. **Check Cloud Function logs**:
   ```bash
   gcloud functions logs read process-audio-upload --limit=50
   ```

3. **Monitor API processing**:
   ```bash
   # Check track status
   curl https://your-api.com/v1/tracks/uuid/status \
     -H "Authorization: Nostr <base64-event>"
   ```

## Troubleshooting

### Common Issues

1. **Cloud Function not triggering**:
   - Check GCS bucket name matches deployment
   - Verify IAM permissions for Cloud Functions service account
   - Check function logs for errors

2. **API webhook fails**:
   - Verify API_BASE_URL is correct and accessible
   - Check WEBHOOK_SECRET matches if configured
   - Monitor API logs for webhook requests

3. **Processing hangs**:
   - Check if ffmpeg is installed in processing environment
   - Verify GCS permissions for reading/writing objects
   - Monitor processing service logs

### Monitoring

Set up Cloud Monitoring alerts for:
- Cloud Function execution failures
- API webhook endpoint errors  
- Processing timeouts
- File upload failures

This setup provides immediate, reliable processing triggered by GCS events rather than polling or delays.