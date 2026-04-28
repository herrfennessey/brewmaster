output "cloud_run_service_account_email" {
  description = "Email of the Cloud Run service account"
  value       = google_service_account.cloud_run_sa.email
}

output "artifact_registry_repository" {
  description = "Artifact Registry Docker repository name"
  value       = google_artifact_registry_repository.docker.name
}

# =============================================================================
# GitHub Actions configuration — paste these into repo Settings
# =============================================================================

output "github_secrets_config" {
  description = "Add these as GitHub repository Secrets (Settings > Secrets and variables > Actions)"
  value = {
    WIF_PROVIDER        = google_iam_workload_identity_pool_provider.github.name
    WIF_SERVICE_ACCOUNT = google_service_account.github_actions.email
  }
}

output "github_variables_config" {
  description = "Add these as GitHub repository Variables (Settings > Secrets and variables > Actions)"
  value = {
    GCP_PROJECT_ID = var.project_id
    GCP_REGION     = var.region
  }
}

output "seed_secrets_commands" {
  description = "Run these once to populate the Secret Manager secrets after terraform apply"
  value       = <<-EOT
    # Anthropic API key
    echo -n "sk-ant-..." | gcloud secrets versions add anthropic-api-key \
      --data-file=- --project=${var.project_id}

    # OpenAI API key (optional — only needed if using AI_PROVIDER=openai)
    echo -n "sk-proj-..." | gcloud secrets versions add openai-api-key \
      --data-file=- --project=${var.project_id}
  EOT
}

output "workload_identity_provider" {
  description = "WIF provider resource name (value for WIF_PROVIDER secret)"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "github_actions_service_account_email" {
  description = "GitHub Actions SA email (value for WIF_SERVICE_ACCOUNT secret)"
  value       = google_service_account.github_actions.email
}
