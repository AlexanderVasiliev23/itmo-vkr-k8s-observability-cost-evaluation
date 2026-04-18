output "cluster_id" {
  value = yandex_kubernetes_cluster.k8s-cluster.id
}

output "cluster_name" {
  value = yandex_kubernetes_cluster.k8s-cluster.name
}

output "get_credentials_cmd" {
  description = "Run this to configure kubectl"
  value       = "yc managed-kubernetes cluster get-credentials ${yandex_kubernetes_cluster.k8s-cluster.name} --external --force"
}
