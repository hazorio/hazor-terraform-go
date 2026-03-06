terraform {
  required_providers {
    hazor = {
      source  = "hazor-cloud/hazor"
      version = "~> 0.1"
    }
  }
}

provider "hazor" {
  endpoint = "https://api.hazor.cloud"
  api_key  = var.hazor_api_key
}

variable "hazor_api_key" {
  description = "API key for the Hazor provider"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "Deployment region"
  type        = string
  default     = "us-east-1"
}

# =============================================================================
# Networking
# =============================================================================

resource "hazor_vpc" "main" {
  name       = "production-vpc"
  cidr_block = "10.0.0.0/16"
}

resource "hazor_subnet" "public_a" {
  name              = "public-a"
  cidr_block        = "10.0.1.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "${var.region}a"
  is_public         = true
}

resource "hazor_subnet" "public_b" {
  name              = "public-b"
  cidr_block        = "10.0.2.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "${var.region}b"
  is_public         = true
}

resource "hazor_subnet" "private_a" {
  name              = "private-a"
  cidr_block        = "10.0.10.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "${var.region}a"
  is_public         = false
}

resource "hazor_subnet" "private_b" {
  name              = "private-b"
  cidr_block        = "10.0.11.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "${var.region}b"
  is_public         = false
}

resource "hazor_nat_gateway" "main" {
  vpc_id    = hazor_vpc.main.id
  subnet_id = hazor_subnet.public_a.id
}

# =============================================================================
# Security Groups
# =============================================================================

resource "hazor_security_group" "lb" {
  name        = "lb-sg"
  description = "Load balancer security group"
  vpc_id      = hazor_vpc.main.id

  ingress_rules = [
    {
      protocol  = "tcp"
      from_port = 80
      to_port   = 80
      cidr      = "0.0.0.0/0"
    },
    {
      protocol  = "tcp"
      from_port = 443
      to_port   = 443
      cidr      = "0.0.0.0/0"
    },
  ]

  egress_rules = [
    {
      protocol  = "-1"
      from_port = 0
      to_port   = 0
      cidr      = "0.0.0.0/0"
    },
  ]
}

resource "hazor_security_group" "app" {
  name        = "app-sg"
  description = "Application instances security group"
  vpc_id      = hazor_vpc.main.id

  ingress_rules = [
    {
      protocol  = "tcp"
      from_port = 8080
      to_port   = 8080
      cidr      = "10.0.0.0/16"
    },
    {
      protocol  = "tcp"
      from_port = 22
      to_port   = 22
      cidr      = "10.0.0.0/16"
    },
  ]

  egress_rules = [
    {
      protocol  = "-1"
      from_port = 0
      to_port   = 0
      cidr      = "0.0.0.0/0"
    },
  ]
}

resource "hazor_security_group" "data" {
  name        = "data-sg"
  description = "Database and cache security group"
  vpc_id      = hazor_vpc.main.id

  ingress_rules = [
    {
      protocol  = "tcp"
      from_port = 5432
      to_port   = 5432
      cidr      = "10.0.0.0/16"
    },
    {
      protocol  = "tcp"
      from_port = 6379
      to_port   = 6379
      cidr      = "10.0.0.0/16"
    },
  ]

  egress_rules = [
    {
      protocol  = "-1"
      from_port = 0
      to_port   = 0
      cidr      = "0.0.0.0/0"
    },
  ]
}

# =============================================================================
# SSH Keys
# =============================================================================

resource "hazor_key_pair" "deploy" {
  name       = "deploy-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

# =============================================================================
# Load Balancer
# =============================================================================

resource "hazor_load_balancer" "web" {
  name    = "web-lb"
  vpc_id  = hazor_vpc.main.id
  lb_type = "application"
}

# =============================================================================
# Compute Instances
# =============================================================================

resource "hazor_instance" "app_1" {
  name          = "app-server-1"
  instance_type = "hz.large"
  image_id      = "ubuntu-22.04"
  vpc_id        = hazor_vpc.main.id
  subnet_id     = hazor_subnet.private_a.id
  key_pair_id   = hazor_key_pair.deploy.id

  user_data = <<-EOF
    #!/bin/bash
    apt-get update && apt-get install -y docker.io
    systemctl enable docker
  EOF
}

resource "hazor_instance" "app_2" {
  name          = "app-server-2"
  instance_type = "hz.large"
  image_id      = "ubuntu-22.04"
  vpc_id        = hazor_vpc.main.id
  subnet_id     = hazor_subnet.private_b.id
  key_pair_id   = hazor_key_pair.deploy.id

  user_data = <<-EOF
    #!/bin/bash
    apt-get update && apt-get install -y docker.io
    systemctl enable docker
  EOF
}

# =============================================================================
# Block Storage
# =============================================================================

resource "hazor_volume" "data" {
  name              = "app-data"
  size_gb           = 100
  volume_type       = "ssd"
  availability_zone = "${var.region}a"
}

# =============================================================================
# Database
# =============================================================================

resource "hazor_database" "main" {
  name           = "production-db"
  engine         = "postgres"
  engine_version = "16"
  instance_class = "db.large"
  storage_gb     = 200
  vpc_id         = hazor_vpc.main.id
}

# =============================================================================
# Redis Cache
# =============================================================================

resource "hazor_redis_instance" "cache" {
  name      = "app-cache"
  memory_mb = 2048
  vpc_id    = hazor_vpc.main.id
  subnet_id = hazor_subnet.private_a.id
}

# =============================================================================
# Object Storage
# =============================================================================

resource "hazor_bucket" "assets" {
  name   = "production-assets"
  region = var.region
}

resource "hazor_bucket" "backups" {
  name   = "production-backups"
  region = var.region
}

# =============================================================================
# DNS
# =============================================================================

resource "hazor_dns_zone" "main" {
  name = "example.com"
}

resource "hazor_dns_record" "root" {
  zone_id     = hazor_dns_zone.main.id
  name        = "@"
  record_type = "A"
  value       = "1.2.3.4"
  ttl         = 300
}

resource "hazor_dns_record" "www" {
  zone_id     = hazor_dns_zone.main.id
  name        = "www"
  record_type = "CNAME"
  value       = "example.com"
  ttl         = 300
}

resource "hazor_dns_record" "api" {
  zone_id     = hazor_dns_zone.main.id
  name        = "api"
  record_type = "A"
  value       = "1.2.3.4"
  ttl         = 60
}

# =============================================================================
# CDN
# =============================================================================

resource "hazor_cdn_distribution" "assets" {
  domain     = "cdn.example.com"
  origin_url = "https://${hazor_bucket.assets.name}.storage.hazor.cloud"
}

# =============================================================================
# Kubernetes Cluster
# =============================================================================

resource "hazor_k8s_cluster" "main" {
  name    = "production-k8s"
  version = "1.29"
  plan_id = "pro-3node"
  vpc_id  = hazor_vpc.main.id
}

# =============================================================================
# Serverless Functions
# =============================================================================

resource "hazor_function" "webhook" {
  name      = "webhook-handler"
  runtime   = "nodejs20"
  handler   = "index.handler"
  memory_mb = 256
  vpc_id    = hazor_vpc.main.id
  subnet_id = hazor_subnet.private_a.id

  environment = {
    DB_HOST     = hazor_database.main.endpoint
    REDIS_HOST  = hazor_redis_instance.cache.host
    NODE_ENV    = "production"
  }
}

# =============================================================================
# Streaming (Kafka)
# =============================================================================

resource "hazor_streaming_cluster" "events" {
  name       = "event-stream"
  vpc_id     = hazor_vpc.main.id
  node_count = 3
}

# =============================================================================
# Container Registry
# =============================================================================

resource "hazor_container_registry" "main" {
  name = "production-registry"
}

# =============================================================================
# PostgresML
# =============================================================================

resource "hazor_postgresml_instance" "ml" {
  name       = "ml-instance"
  plan_name  = "professional"
  vcpu_count = 8
  memory_mb  = 32768
  storage_gb = 500
  region     = var.region
}

# =============================================================================
# NoSQL (MongoDB)
# =============================================================================

resource "hazor_nosql_instance" "documents" {
  name       = "doc-store"
  engine     = "mongodb"
  vcpus      = 4
  memory_mb  = 8192
  storage_gb = 100
  vpc_id     = hazor_vpc.main.id
  subnet_id  = hazor_subnet.private_a.id
}

# =============================================================================
# Snapshots
# =============================================================================

resource "hazor_snapshot" "data_backup" {
  name        = "data-snapshot"
  volume_id   = hazor_volume.data.id
  description = "Initial backup of application data volume"
}

# =============================================================================
# CI/CD Runner
# =============================================================================

resource "hazor_runner" "ci" {
  org_id = "org-production"
  labels = ["linux", "docker", "large"]
}

# =============================================================================
# Bun App
# =============================================================================

resource "hazor_bun_app" "api" {
  name      = "api-service"
  vcpus     = 2
  memory_mb = 1024
}

# =============================================================================
# Supabase
# =============================================================================

resource "hazor_supabase_instance" "backend" {
  name            = "app-backend"
  plan_vcpu       = 4
  plan_memory_mb  = 8192
  plan_storage_gb = 50
  region          = var.region
}

# =============================================================================
# Outputs
# =============================================================================

output "vpc_id" {
  value = hazor_vpc.main.id
}

output "load_balancer_id" {
  value = hazor_load_balancer.web.id
}

output "database_endpoint" {
  value     = hazor_database.main.endpoint
  sensitive = true
}

output "redis_host" {
  value = hazor_redis_instance.cache.host
}

output "redis_port" {
  value = hazor_redis_instance.cache.port
}

output "k8s_endpoint" {
  value     = hazor_k8s_cluster.main.endpoint
  sensitive = true
}

output "cdn_domain" {
  value = hazor_cdn_distribution.assets.domain
}

output "nosql_host" {
  value = hazor_nosql_instance.documents.host
}

output "registry_name" {
  value = hazor_container_registry.main.name
}
