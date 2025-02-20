data "mongodbatlas_cluster" "singular" {
  # data source content is kep, only data source type is changed - singular
  project_id = mongodbatlas_advanced_cluster.example.project_id
  name       = mongodbatlas_advanced_cluster.example.name
  depends_on = [mongodbatlas_privatelink_endpoint_service.example_endpoint]
}

data "mongodbatlas_clusters" "plural" {
  # data source content is kep, only data source type is changed - plural
  project_id = mongodbatlas_advanced_cluster.example.project_id
}
