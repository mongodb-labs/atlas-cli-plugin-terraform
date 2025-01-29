resource "resource1" "res1" {
  name = "name1"
}

resource "mongodbatlas_advanced_cluster" "cluster1" {
  name = "name2"
}

data "resource2" "res2" {
  name = "name3"
}

data "mongodbatlas_cluster" "cluster2" {
  name = "name4"
}
