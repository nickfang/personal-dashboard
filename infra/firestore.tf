# Enable the Firestore API
resource "google_project_service" "firestore" {
  service            = "firestore.googleapis.com"
  disable_on_destroy = false
}

# Create the Firestore Database (Native Mode)
resource "google_firestore_database" "database" {
  # Depends on the API being enabled
  depends_on = [google_project_service.firestore]

  project     = var.project_id
  name        = "(default)"
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}
