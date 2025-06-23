# Cloud Build Trigger Setup Instructions

This guide will walk you through setting up automatic deployments to Cloud Run when commits are pushed to the main branch of the GitHub repository.

## Prerequisites

1. A Google Cloud Project with billing enabled
2. GitHub repository: https://github.com/wavlake/api
3. gcloud CLI installed and configured
4. Appropriate permissions in both GitHub and Google Cloud

## Step 1: Enable Required APIs

Run the following commands to enable the necessary Google Cloud APIs:

```bash
# Set your project ID
export PROJECT_ID=your-project-id

# Enable required APIs
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable containerregistry.googleapis.com
gcloud services enable secretmanager.googleapis.com
```

## Step 2: Create Service Account

Create a service account for the Cloud Run service:

```bash
# Create service account
gcloud iam service-accounts create api-service \
    --display-name="API Service Account"

# Grant necessary permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:api-service@$PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/run.invoker"
```

## Step 3: Configure Cloud Build Service Account Permissions

Grant Cloud Build the necessary permissions:

```bash
# Get Cloud Build service account
export CLOUD_BUILD_SA=$(gcloud projects describe $PROJECT_ID --format='value(projectNumber)')@cloudbuild.gserviceaccount.com

# Grant Cloud Run Admin permission
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/run.admin"

# Grant Service Account User permission
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/iam.serviceAccountUser"

# Grant Container Registry permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$CLOUD_BUILD_SA" \
    --role="roles/storage.admin"
```

## Step 4: Connect GitHub Repository

### Option A: Using Cloud Console (Recommended)

1. Go to the [Cloud Build Triggers page](https://console.cloud.google.com/cloud-build/triggers)
2. Click "Connect Repository"
3. Select "GitHub (Cloud Build GitHub App)"
4. Authenticate with GitHub if prompted
5. Select "wavlake" organization
6. Select "api" repository
7. Click "Connect"

### Option B: Using GitHub App

1. Install the [Google Cloud Build GitHub App](https://github.com/marketplace/google-cloud-build)
2. Configure it for the wavlake/api repository
3. Grant necessary permissions

## Step 5: Create Cloud Build Trigger

### Using Cloud Console:

1. Go to [Cloud Build Triggers](https://console.cloud.google.com/cloud-build/triggers)
2. Click "Create Trigger"
3. Configure the trigger:
   - **Name**: `api-main-deploy`
   - **Description**: "Deploy API to Cloud Run on push to main"
   - **Event**: Push to a branch
   - **Source**: Select your connected repository (wavlake/api)
   - **Branch**: `^main$`
   - **Build Configuration**: Cloud Build configuration file
   - **Location**: `/cloudbuild-trigger.yaml`
4. Click "Create"

### Using gcloud CLI:

```bash
gcloud builds triggers create github \
    --repo-name=api \
    --repo-owner=wavlake \
    --branch-pattern="^main$" \
    --build-config=cloudbuild-trigger.yaml \
    --name=api-main-deploy \
    --description="Deploy API to Cloud Run on push to main"
```

## Step 6: Configure Substitution Variables (Optional)

You can customize the deployment by modifying substitution variables:

1. Go to your trigger in Cloud Console
2. Click "Edit"
3. Under "Substitution variables", you can override defaults:
   - `_REGION`: Deployment region (default: us-central1)
   - `_MIN_INSTANCES`: Minimum instances (default: 1)
   - `_MAX_INSTANCES`: Maximum instances (default: 100)
   - `_MEMORY`: Memory allocation (default: 512Mi)
   - `_CPU`: CPU allocation (default: 1)
   - `_TIMEOUT`: Request timeout (default: 60s)
   - `_CONCURRENCY`: Max concurrent requests per instance (default: 80)

## Step 7: Test the Trigger

1. Make a small change to your repository
2. Commit and push to the main branch:
   ```bash
   git add .
   git commit -m "Test Cloud Build trigger"
   git push origin main
   ```
3. Monitor the build:
   - Go to [Cloud Build History](https://console.cloud.google.com/cloud-build/builds)
   - Check the build logs
   - Verify deployment in [Cloud Run](https://console.cloud.google.com/run)

## Step 8: Set Up Notifications (Optional)

### Email Notifications:
1. Go to Cloud Build Settings
2. Enable "Email notifications"
3. Configure notification preferences

### Slack Notifications:
1. Create a Pub/Sub topic:
   ```bash
   gcloud pubsub topics create cloud-builds
   ```
2. Configure Cloud Build to publish to the topic
3. Set up a Cloud Function to send Slack notifications

## Environment Variables

The following environment variables are automatically set during deployment:
- `COMMIT_SHA`: Git commit SHA of the deployment
- `ENV`: Environment name (production)
- `PORT`: Port number (8080)

## Security Best Practices

1. **Least Privilege**: Only grant necessary permissions
2. **Secret Management**: Use Secret Manager for sensitive data:
   ```bash
   # Create a secret
   echo -n "your-secret-value" | gcloud secrets create api-secret --data-file=-
   
   # Grant access to the service account
   gcloud secrets add-iam-policy-binding api-secret \
       --member="serviceAccount:api-service@$PROJECT_ID.iam.gserviceaccount.com" \
       --role="roles/secretmanager.secretAccessor"
   ```
3. **VPC Connector**: For private resources, configure a VPC connector
4. **Binary Authorization**: Enable binary authorization for additional security

## Monitoring and Debugging

### View Build Logs:
```bash
gcloud builds log [BUILD_ID]
```

### View Cloud Run Logs:
```bash
gcloud run services logs read api --region=us-central1
```

### Monitor Service:
```bash
gcloud run services describe api --region=us-central1
```

## Rollback Procedure

If you need to rollback to a previous version:

```bash
# List revisions
gcloud run revisions list --service=api --region=us-central1

# Route traffic to a previous revision
gcloud run services update-traffic api \
    --to-revisions=[REVISION_NAME]=100 \
    --region=us-central1
```

## Troubleshooting

### Common Issues:

1. **Permission Denied**: Ensure Cloud Build service account has necessary permissions
2. **Build Timeout**: Increase timeout in cloudbuild-trigger.yaml
3. **Container Registry Access**: Verify storage.admin role is granted
4. **Service Account Issues**: Check service account exists and has correct permissions

### Debug Commands:
```bash
# Check trigger status
gcloud builds triggers list

# View recent builds
gcloud builds list --limit=5

# Check Cloud Run service
gcloud run services describe api --region=us-central1

# Test service endpoint
curl https://api-[HASH]-uc.a.run.app/heartbeat
```

## Additional Resources

- [Cloud Build Documentation](https://cloud.google.com/build/docs)
- [Cloud Run Documentation](https://cloud.google.com/run/docs)
- [Cloud Build Pricing](https://cloud.google.com/build/pricing)
- [Cloud Run Pricing](https://cloud.google.com/run/pricing)

## Support

For issues or questions:
1. Check Cloud Build logs for detailed error messages
2. Review IAM permissions
3. Consult Google Cloud documentation
4. Contact your cloud administrator