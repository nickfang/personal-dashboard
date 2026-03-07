resource "google_project_service" "run" {
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "artifactregistry" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudscheduler" {
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "cloudbuild" {
  service            = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "secretmanager" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "firestore" {
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "iamcredentials" {
  service            = "iamcredentials.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "weather" {
  service            = "weather.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "pollen" {
  service            = "pollen.googleapis.com"
  disable_on_destroy = false
}

resource "time_sleep" "api_propagation" {
  create_duration = "60s"

  depends_on = [
    google_project_service.run,
    google_project_service.artifactregistry,
    google_project_service.cloudscheduler,
    google_project_service.cloudbuild,
    google_project_service.secretmanager,
    google_project_service.firestore,
    google_project_service.iamcredentials,
    google_project_service.weather,
    google_project_service.pollen,
  ]
}

resource "google_artifact_registry_repository" "repo" {
  provider      = google-beta
  project       = var.project_id
  location      = var.region
  repository_id = "personal-dashboard"
  description   = "Docker repository for Personal Dashboard services"
  format        = "DOCKER"

  depends_on = [time_sleep.api_propagation]
}
