# GCS Bucket Setup Guide

This guide helps you create and configure a Google Cloud Storage bucket for the Wavlake audio upload system.

## Quick Setup (Automated)

### Prerequisites

1. **Google Cloud CLI installed**:
   ```bash
   # Install gcloud CLI if not already installed
   curl https://sdk.cloud.google.com | bash
   exec -l $SHELL
   gcloud init
   ```

2. **Authenticate and set project**:
   ```bash
   gcloud auth login
   gcloud config set project your-project-id
   export GOOGLE_CLOUD_PROJECT=your-project-id
   ```

### Create Bucket

Run the automated setup script:

```bash
# Option 1: Let script prompt for bucket name
./setup-gcs-bucket.sh

# Option 2: Set bucket name first
export GCS_BUCKET_NAME=wavlake-audio-uploads-12345
./setup-gcs-bucket.sh
```

The script will:
- ✅ Create the bucket with proper configuration
- ✅ Set up directory structure (`tracks/original/`, `tracks/compressed/`)
- ✅ Configure permissions (private originals, public compressed)
- ✅ Set lifecycle policies for cleanup
- ✅ Enable CORS for browser uploads
- ✅ Verify the setup

## Manual Setup (Step by Step)

If you prefer to set up manually:

### 1. Create the Bucket

```bash
export GOOGLE_CLOUD_PROJECT=your-project-id
export GCS_BUCKET_NAME=your-unique-bucket-name
export GCS_REGION=us-central1

# Create bucket
gsutil mb -p $GOOGLE_CLOUD_PROJECT -c STANDARD -l $GCS_REGION gs://$GCS_BUCKET_NAME
```

### 2. Set Up Directory Structure

```bash
# Create directory placeholders
echo "placeholder" | gsutil cp - gs://$GCS_BUCKET_NAME/tracks/original/.placeholder
echo "placeholder" | gsutil cp - gs://$GCS_BUCKET_NAME/tracks/compressed/.placeholder

# Remove placeholders
gsutil rm gs://$GCS_BUCKET_NAME/tracks/original/.placeholder
gsutil rm gs://$GCS_BUCKET_NAME/tracks/compressed/.placeholder
```

### 3. Configure Permissions

```bash
# Remove default public access
gsutil iam ch -d allUsers:objectViewer gs://$GCS_BUCKET_NAME

# Allow public read for compressed files only
gsutil iam ch allUsers:objectViewer gs://$GCS_BUCKET_NAME/tracks/compressed/*
```

### 4. Set Lifecycle Policy

```bash
cat > lifecycle.json << EOF
{
  "lifecycle": {
    "rule": [
      {
        "condition": {"age": 1, "isLive": false},
        "action": {"type": "Delete"}
      },
      {
        "condition": {"age": 7},
        "action": {"type": "AbortIncompleteMultipartUpload"}
      }
    ]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://$GCS_BUCKET_NAME
rm lifecycle.json
```

### 5. Configure CORS

```bash
cat > cors.json << EOF
[
  {
    "origin": ["*"],
    "method": ["GET", "PUT", "POST", "HEAD"],
    "responseHeader": ["Content-Type", "ETag"],
    "maxAgeSeconds": 3600
  }
]
EOF

gsutil cors set cors.json gs://$GCS_BUCKET_NAME
rm cors.json
```

## Bucket Configuration Details

### Directory Structure
```
gs://your-bucket-name/
├── tracks/
│   ├── original/       # Private - original uploads
│   │   ├── uuid1.flac
│   │   ├── uuid2.wav
│   │   └── uuid3.mp3
│   └── compressed/     # Public - streaming optimized
│       ├── uuid1.mp3
│       ├── uuid2.mp3
│       └── uuid3.mp3
```

### Permissions
- **Original files** (`tracks/original/`): Private access only
- **Compressed files** (`tracks/compressed/`): Public read access for streaming
- **API service account**: Full access for upload/download operations

### Lifecycle Policies
- **Incomplete uploads**: Cleaned up after 7 days
- **Non-live object versions**: Deleted after 1 day
- **Cost optimization**: Prevents storage charges for failed uploads

### CORS Configuration
- **Origins**: Allow all origins (customize for production)
- **Methods**: GET, PUT, POST, HEAD
- **Headers**: Content-Type, ETag
- **Cache**: 1 hour max age

## Bucket Naming Best Practices

### ✅ Good Bucket Names
- `wavlake-audio-prod-us-central1`
- `mycompany-tracks-staging-2024`
- `audio-uploads-12345678`

### ❌ Avoid
- Names with spaces or special characters
- Names starting with "goog" or containing "google"
- Names that look like IP addresses
- Very short names (likely taken)

### Tips
- Include your company/project name
- Add environment suffix (`-prod`, `-staging`)
- Add region for clarity
- Add random numbers for uniqueness

## Verification

After setup, verify your bucket:

```bash
# Check bucket exists
gsutil ls gs://$GCS_BUCKET_NAME

# Check structure
gsutil ls gs://$GCS_BUCKET_NAME/tracks/

# Check permissions
gsutil iam get gs://$GCS_BUCKET_NAME

# Check lifecycle
gsutil lifecycle get gs://$GCS_BUCKET_NAME

# Check CORS
gsutil cors get gs://$GCS_BUCKET_NAME
```

## Security Considerations

### Service Account Access
The API needs a service account with these permissions:
- `Storage Object Admin` on the bucket
- `Storage Object Creator` for uploads
- `Storage Object Viewer` for downloads

### Public Access
Only compressed files are publicly accessible:
- Original files remain private
- Compressed files are readable by anyone with the URL
- No listing permissions for security

### Network Security
- CORS configured for browser uploads
- Consider restricting origins in production
- Monitor access logs for unusual activity

## Costs

### Storage Costs (US regions)
- **Standard Storage**: ~$0.020/GB/month
- **Nearline Storage**: ~$0.010/GB/month (if using lifecycle)

### Operation Costs
- **Write operations**: ~$0.05/1000 operations
- **Read operations**: ~$0.004/1000 operations

### Bandwidth
- **Egress to internet**: $0.12/GB (first 1TB free per month)
- **Egress within GCP**: Free

### Cost Optimization Tips
- Use lifecycle policies to move old files to cheaper storage
- Monitor usage with Cloud Monitoring
- Set up billing alerts
- Consider Nearline storage for archival files

## Troubleshooting

### Common Issues

1. **Bucket name already exists**:
   ```bash
   # Try with added suffix
   export GCS_BUCKET_NAME=your-bucket-name-$(date +%s)
   ```

2. **Permission denied**:
   ```bash
   # Check authentication
   gcloud auth list
   gcloud auth application-default login
   ```

3. **CORS not working**:
   ```bash
   # Verify CORS settings
   gsutil cors get gs://$GCS_BUCKET_NAME
   ```

4. **Lifecycle not applying**:
   ```bash
   # Check lifecycle policy
   gsutil lifecycle get gs://$GCS_BUCKET_NAME
   ```

## Next Steps

After bucket creation:

1. **Update API configuration**:
   ```bash
   export GCS_BUCKET_NAME=your-bucket-name
   ```

2. **Deploy Cloud Function**:
   ```bash
   cd cloud-function/
   ./deploy.sh
   ```

3. **Test the upload flow**:
   ```bash
   # Test with API endpoints
   curl -X POST https://your-api.com/v1/tracks/nostr
   ```

Your GCS bucket is now ready for the Wavlake audio upload system!