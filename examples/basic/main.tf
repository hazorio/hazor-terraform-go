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

# --- VPC ---
resource "hazor_vpc" "main" {
  name       = "main-vpc"
  cidr_block = "10.0.0.0/16"
}

# --- Subnets ---
resource "hazor_subnet" "public" {
  name              = "public-subnet"
  cidr_block        = "10.0.1.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "us-east-1a"
  is_public         = true
}

resource "hazor_subnet" "private" {
  name              = "private-subnet"
  cidr_block        = "10.0.2.0/24"
  vpc_id            = hazor_vpc.main.id
  availability_zone = "us-east-1a"
  is_public         = false
}

# --- Key Pair ---
resource "hazor_key_pair" "deploy" {
  name       = "deploy-key"
  public_key = file("~/.ssh/id_ed25519.pub")
}

# --- Security Group ---
resource "hazor_security_group" "web" {
  name        = "web-sg"
  description = "Allow HTTP and SSH"
  vpc_id      = hazor_vpc.main.id

  ingress_rules = [
    {
      protocol  = "tcp"
      from_port = 22
      to_port   = 22
      cidr      = "0.0.0.0/0"
    },
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

# --- Instance ---
resource "hazor_instance" "web" {
  name          = "web-server"
  instance_type = "hz.medium"
  image_id      = "ubuntu-22.04"
  vpc_id        = hazor_vpc.main.id
  subnet_id     = hazor_subnet.public.id
  key_pair_id   = hazor_key_pair.deploy.id

  user_data = <<-EOF
    #!/bin/bash
    apt-get update
    apt-get install -y nginx
    systemctl enable nginx
    systemctl start nginx
  EOF
}

# --- Elastic IP ---
resource "hazor_elastic_ip" "web" {
  region      = "us-east-1"
  instance_id = hazor_instance.web.id
}

# --- Outputs ---
output "instance_public_ip" {
  value = hazor_instance.web.public_ip
}

output "elastic_ip" {
  value = hazor_elastic_ip.web.ip_address
}

output "vpc_id" {
  value = hazor_vpc.main.id
}
