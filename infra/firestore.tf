# Create the Firestore Database (Native Mode)
resource "google_firestore_database" "database" {
  # Depends on the API being enabled (managed in main.tf)
  depends_on = [google_project_service.firestore]

  project     = var.project_id
  name        = "weather-log"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}
