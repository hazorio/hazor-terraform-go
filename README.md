# Terraform Provider for Hazor

Manage [Hazor Cloud](https://hazor.cloud) infrastructure using Terraform.

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Go](https://go.dev/dl/) >= 1.21 (to build the provider)

## Building

```bash
go build -o terraform-provider-hazor
```

## Configuration

```hcl
terraform {
  required_providers {
    hazor = {
      source  = "hazor-cloud/hazor"
      version = "~> 0.1"
    }
  }
}

provider "hazor" {
  endpoint = "https://api.hazor.cloud"  # or HAZOR_ENDPOINT env var
  api_key  = var.hazor_api_key          # or HAZOR_API_KEY env var
}
```

| Argument   | Environment Variable | Description                   |
|------------|----------------------|-------------------------------|
| `endpoint` | `HAZOR_ENDPOINT`     | Hazor API base URL            |
| `api_key`  | `HAZOR_API_KEY`      | API key for authentication    |

## Resources

### Compute
| Resource | Description |
|----------|-------------|
| `hazor_instance` | Virtual machine instance |
| `hazor_key_pair` | SSH key pair |
| `hazor_snapshot` | Volume snapshot |
| `hazor_bun_app` | Bun.js application |
| `hazor_function` | Serverless function |
| `hazor_runner` | CI/CD runner |

### Networking
| Resource | Description |
|----------|-------------|
| `hazor_vpc` | Virtual Private Cloud |
| `hazor_subnet` | VPC subnet |
| `hazor_security_group` | Firewall rules |
| `hazor_elastic_ip` | Static public IP |
| `hazor_nat_gateway` | NAT gateway |
| `hazor_load_balancer` | Application/network load balancer |
| `hazor_dns_zone` | DNS zone |
| `hazor_dns_record` | DNS record |
| `hazor_cdn_distribution` | CDN distribution |

### Storage
| Resource | Description |
|----------|-------------|
| `hazor_volume` | Block storage volume |
| `hazor_bucket` | S3-compatible object storage bucket |
| `hazor_container_registry` | Container image registry |

### Databases & Data
| Resource | Description |
|----------|-------------|
| `hazor_database` | Managed PostgreSQL cluster |
| `hazor_redis_instance` | Managed Redis instance |
| `hazor_nosql_instance` | Managed ScyllaDB instance |
| `hazor_postgresml_instance` | Managed PostgresML instance |
| `hazor_streaming_cluster` | Managed Redpanda streaming cluster |

### Platforms
| Resource | Description |
|----------|-------------|
| `hazor_k8s_cluster` | Managed Kubernetes (k3s) cluster |
| `hazor_supabase_instance` | Managed Supabase instance |

## Quick Start

```hcl
resource "hazor_vpc" "main" {
  name       = "main-vpc"
  cidr_block = "10.0.0.0/16"
}

resource "hazor_subnet" "public" {
  name              = "public-subnet"
  cidr_block        = "10.0.1.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "us-east-1a"
  is_public         = true
}

resource "hazor_security_group" "web" {
  name   = "web-sg"
  vpc_id = hazor_vpc.main.id

  ingress_rules = [
    { protocol = "tcp", from_port = 22,  to_port = 22,  cidr = "0.0.0.0/0" },
    { protocol = "tcp", from_port = 80,  to_port = 80,  cidr = "0.0.0.0/0" },
    { protocol = "tcp", from_port = 443, to_port = 443, cidr = "0.0.0.0/0" },
  ]

  egress_rules = [
    { protocol = "-1", from_port = 0, to_port = 0, cidr = "0.0.0.0/0" },
  ]
}

resource "hazor_instance" "web" {
  name          = "web-server"
  instance_type = "hz.medium"
  image_id      = "ubuntu-22.04"
  vpc_id        = hazor_vpc.main.id
  subnet_id     = hazor_subnet.public.id
}
```

See [`examples/`](examples/) for more complete configurations.

## Importing Existing Resources

All resources support `terraform import`:

```bash
terraform import hazor_instance.web <instance-uuid>
terraform import hazor_vpc.main <vpc-uuid>
```

## License

MPL-2.0
