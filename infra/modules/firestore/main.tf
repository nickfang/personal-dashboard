resource "google_firestore_database" "database" {
  for_each = toset(var.database_ids)

  project     = var.project_id
  name        = each.value
  location_id = var.region
  type        = "FIRESTORE_NATIVE"
}
