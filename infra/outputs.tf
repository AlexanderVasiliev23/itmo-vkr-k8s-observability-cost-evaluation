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

output "runner_ip" {
  description = "Public IP of the obs-bench runner VM"
  value       = yandex_compute_instance.runner.network_interface[0].nat_ip_address
}

output "runner_ssh_cmd" {
  description = "SSH command to connect to the runner VM"
  value       = "ssh ubuntu@${yandex_compute_instance.runner.network_interface[0].nat_ip_address}"
}
