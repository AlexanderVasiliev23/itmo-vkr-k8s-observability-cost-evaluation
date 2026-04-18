variable "yc_token" {
  description = "Yandex Cloud IAM token (yc iam create-token)"
  type        = string
  sensitive   = true
}

variable "cloud_id" {
  description = "Yandex Cloud cloud ID"
  type        = string
}

variable "folder_id" {
  description = "Yandex Cloud folder ID"
  type        = string
}

variable "zone" {
  description = "Yandex Cloud availability zone"
  type        = string
  default     = "ru-central1-a"
}

variable "cluster_name" {
  description = "Kubernetes cluster name"
  type        = string
  default     = "obs-bench-cluster"
}

variable "node_group_name" {
  description = "Kubernetes node group name"
  type        = string
  default     = "obs-bench-nodes"
}

variable "k8s_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.32"
}

variable "sa_name" {
  description = "Service account name"
  type        = string
  default     = "obs-bench-sa"
}

variable "node_cores" {
  description = "Number of CPU cores per node"
  type        = number
  default     = 8
}

variable "node_memory_gb" {
  description = "RAM per node in GiB"
  type        = number
  default     = 16
}

variable "node_disk_gb" {
  description = "Boot disk size per node in GiB"
  type        = number
  default     = 100
}

variable "node_count" {
  description = "Number of nodes in the node group"
  type        = number
  default     = 1
}
