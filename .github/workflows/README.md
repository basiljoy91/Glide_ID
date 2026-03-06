# CI/CD Pipelines Documentation

This directory contains GitHub Actions workflows for automated testing, building, and deployment of the Enterprise Attendance System.

## Workflows Overview

### 1. `deploy-ui.yml` - Frontend Deployment
**Triggers:**
- Push to `main` or `develop` branches when `frontend-nextjs/**` changes
- Manual workflow dispatch

**Jobs:**
1. **Lint**: Runs ESLint and TypeScript checks
2. **Build**: Builds Next.js application
3. **Deploy**: Deploys to Vercel

**Path Filtering:**
```yaml
paths:
  - 'frontend-nextjs/**'
  - '.github/workflows/deploy-ui.yml'
```

**Required Secrets:**
- `VERCEL_ORG_ID`
- `VERCEL_PROJECT_ID`
- `VERCEL_TOKEN`
- `NEXT_PUBLIC_API_URL`
- `NEXT_PUBLIC_AI_SERVICE_URL`
- `NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY`

### 2. `deploy-go.yml` - Go Backend Deployment
**Triggers:**
- Push to `main` or `develop` branches
- Manual workflow dispatch

**Jobs:**
1. **Test**: Runs Go tests and linter
2. **Build**: Builds binaries for multiple platforms (Linux, macOS, Windows)
3. **Deploy Render**: Deploys to Render (primary)
4. **Deploy Koyeb**: Alternative deployment option (disabled by default)
5. **Docker Build**: Builds and pushes Docker image

**Required Secrets:**
- `RENDER_SERVICE_ID`
- `RENDER_API_KEY`
- `DATABASE_URL`
- `JWT_SECRET`
- `AI_SERVICE_URL`
- `AI_SERVICE_API_KEY`
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`

### 3. `deploy-ai.yml` - Python AI Service Deployment
**Triggers:**
- Push to `main` or `develop` branches
- Manual workflow dispatch

**Jobs:**
1. **Test**: Runs Python tests and linter
2. **Build**: Builds Docker image
3. **Deploy Hugging Face**: Deploys to Hugging Face Spaces
4. **Deploy VPS**: Alternative VPS deployment (disabled by default)
5. **Docker Compose**: Creates docker-compose configuration

**Required Secrets:**
- `HUGGINGFACE_TOKEN`
- `DOCKER_USERNAME`
- `DOCKER_PASSWORD`
- `DATABASE_URL`
- `AI_SERVICE_API_KEY`
- `ENCRYPTION_KEY`
- `VPS_HOST` (optional)
- `VPS_USERNAME` (optional)
- `VPS_SSH_KEY` (optional)

### 4. `ci.yml` - Continuous Integration
**Triggers:**
- Pull requests to `main` or `develop`
- Manual workflow dispatch

**Jobs:**
- Runs tests and linters for all services
- Does not deploy, only validates code quality

## Trigger Strategy

Deployment workflows are configured to run on every push to `main` and `develop` so deployment is predictable and visible in the Actions tab.

### Example Scenarios

1. Push to `main`: all deployment workflows can run
2. Push to `develop`: all deployment workflows can run
3. Manual run: use `workflow_dispatch` from Actions tab

## Setting Up Secrets

### GitHub Secrets Configuration

1. Go to repository Settings → Secrets and variables → Actions
2. Add the following secrets:

#### Frontend Secrets
```
VERCEL_ORG_ID=your-org-id
VERCEL_PROJECT_ID=your-project-id
VERCEL_TOKEN=your-vercel-token
NEXT_PUBLIC_API_URL=https://api.yourdomain.com
NEXT_PUBLIC_AI_SERVICE_URL=https://ai.yourdomain.com
NEXT_PUBLIC_ENCRYPTION_PUBLIC_KEY=your-public-key
```

#### Backend Secrets
```
RENDER_SERVICE_ID=your-service-id
RENDER_API_KEY=your-render-api-key
DATABASE_URL=postgresql://...
JWT_SECRET=your-jwt-secret
AI_SERVICE_URL=https://ai.yourdomain.com
AI_SERVICE_API_KEY=your-ai-api-key
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-password
```

#### AI Service Secrets
```
HUGGINGFACE_TOKEN=your-hf-token
DOCKER_USERNAME=your-docker-username
DOCKER_PASSWORD=your-docker-password
DATABASE_URL=postgresql://...
AI_SERVICE_API_KEY=your-ai-api-key
ENCRYPTION_KEY=your-encryption-key
```

## Deployment Platforms

### Frontend: Vercel
- Automatic deployments on push
- Preview deployments for PRs
- Environment variables managed in Vercel dashboard

### Backend: Render / Koyeb
- Render: Primary deployment platform
- Koyeb: Alternative option (configure in workflow)
- Docker: Container-based deployment

### AI Service: Hugging Face Spaces / VPS
- Hugging Face Spaces: Free tier available
- VPS: Self-hosted option
- Docker: Container-based deployment

## Workflow Best Practices

1. **Path Filtering**: Always include workflow file in paths to allow manual triggers
2. **Secrets**: Never commit secrets to repository
3. **Artifacts**: Upload build artifacts for debugging
4. **Caching**: Use dependency caching to speed up builds
5. **Matrix Builds**: Build for multiple platforms when needed

## Troubleshooting

### Workflow Not Triggering
- Verify branch name is `main` or `develop`
- Verify Actions are enabled in repository settings
- Check workflow file syntax

### Deployment Fails
- Verify all secrets are set correctly
- Check deployment platform logs
- Review build artifacts

### Build Timeout
- Increase timeout in workflow file
- Optimize build process
- Use caching for dependencies

## Manual Deployment

All workflows support manual triggering:
1. Go to Actions tab
2. Select workflow
3. Click "Run workflow"
4. Select branch and click "Run workflow"

## Monitoring

- Check Actions tab for workflow status
- Set up notifications for failed deployments
- Review deployment logs for errors

## Security Considerations

- All secrets stored in GitHub Secrets
- No secrets in workflow files
- Docker images scanned for vulnerabilities
- Dependencies updated regularly
