# Disaster Recovery Runbook

## Overview

How to rebuild the entire infrastructure from scratch using Terraform modules in `infra/`.

## Prerequisites

- Google Cloud SDK (`gcloud`) authenticated
- Terraform 1.5+
- Access to create/manage GCP projects
- GitHub repo admin access (for deployment environments)

## Steps

### 1. Create GCP Project

- Create a new GCP project manually in the GCP console
- Link a billing account to the project

### 2. Configure Terraform

- Navigate to the environment directory: `cd infra/prod/` (or `infra/staging/`)
- Copy `terraform.tfvars.example` to `terraform.tfvars`
- Fill in `project_id` with the new GCP project ID

### 3. Initialize and Apply Terraform

```bash
terraform init
terraform apply
```

**Note:** The first apply will partially fail:

- The 5 `null_resource` bootstrap builds run on every fresh apply (they have no GCP-side state and can't be imported). They rebuild Docker images via Cloud Build.
- The Cloud Run **jobs** (weather-collector, pollen-collector) will fail because they reference a Secret Manager secret version that doesn't exist yet. Services and all other resources will be created successfully.
- If bootstrap builds fail with PERMISSION_DENIED, wait 60 seconds for API propagation and re-run `terraform apply`.

### 4. Add Secret Values

**Required before collectors will function.** Terraform creates a placeholder secret version so Cloud Run jobs can be created, but the collectors will fail at runtime until the real API key is added.

1. In the GCP console, go to **APIs & Services → Credentials** and create an API key
2. Add the key to Secret Manager:

```bash
echo -n "YOUR_API_KEY" | gcloud secrets versions add google-maps-api-key --data-file=- --project=<project_id>
```

The key is used by the weather-collector (`weather.googleapis.com`) and pollen-collector (`pollen.googleapis.com`) Cloud Run jobs.

### 5. Complete Terraform Apply

Run `terraform apply` again to create the Cloud Run jobs that failed in step 3:

```bash
terraform apply
```

### 6. Configure Custom Domain Mapping

After Terraform apply, the domain mapping module outputs the DNS records needed.

1. **Verify domain ownership** (one-time, if not already done):
   - Visit `https://www.google.com/webmasters/verification/verification?domain=yourdomain.com` (replace with your actual domain)
   - Add the TXT record Google provides to your DNS registrar
   - Click **Verify** once DNS has propagated
   - Verify at the root domain level to cover all subdomains

2. **Add DNS record at your registrar:**

   | Type | Name | Value |
   |------|------|-------|
   | CNAME | `api-staging` | `ghs.googlehosted.com.` |

3. **Wait for TLS certificate provisioning** (15–60 minutes). Check status:

   ```bash
   gcloud run domain-mappings describe --domain api-staging.<yourdomain>.com --region us-central1
   ```

   Look for `certificateStatus: ACTIVE`.

4. **Re-run `terraform apply`** if the domain mapping was in a failed state due to missing DNS verification.

### 7. Record Terraform Outputs

```bash
terraform output
```

Save these values — they're needed for GitHub deployment environments:

- `github_actions_sa_email`
- `workload_identity_provider_name`
- `artifact_registry_url`

### 8. Configure GitHub Deployment Environments

In the GitHub repo, go to **Settings -> Environments** and create/update the environment (`staging` or `production`).

Add these **environment variables** (not secrets):

| Variable | Value |
|---|---|
| `GCP_PROJECT_ID` | Your GCP project ID |
| `GCP_REGION` | `us-central1` |
| `WIF_PROVIDER` | Value from `workload_identity_provider_name` output |
| `GCP_SA_EMAIL` | Value from `github_actions_sa_email` output |

Optionally add required reviewers to the `production` environment.

### 9. Verify

Push a change to `main` and confirm the staging deploy workflow succeeds. Create a release to confirm the prod deploy workflow succeeds.

## Infrastructure Layout

```
infra/
  modules/          # Shared Terraform modules
    foundation/     # API enables + Artifact Registry
    firestore/      # Firestore databases
    secrets/        # Secret Manager (empty containers + placeholder versions)
    cloud-run-job/  # Collector services (Cloud Run Jobs + Scheduler)
    cloud-run-provider/   # Internal gRPC services
    cloud-run-aggregator/ # Public BFF (Dashboard API)
    cloud-run-domain-mapping/ # Custom domain mapping for Cloud Run services
    github-oidc/    # GitHub Actions OIDC authentication
  staging/          # Staging environment Terraform root
  prod/             # Production environment Terraform root
```

## Notes

- State is local (not remote). Each environment has its own `terraform.tfstate` in its directory.
- The `terraform.tfstate` and `terraform.tfvars` files are gitignored.
- The "rebuild from scratch" DR principle means we don't need remote state — infrastructure can always be recreated from the Terraform code.
