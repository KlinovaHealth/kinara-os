terraform {
  required_version = ">= 1.6"

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.39"
    }
  }

  # State stored in DigitalOcean Spaces (S3-compatible)
  backend "s3" {
    endpoint                    = "https://fra1.digitaloceanspaces.com"
    bucket                      = "kinara-tf-state"
    key                         = "prod/kinara-os.tfstate"
    region                      = "us-east-1"  # required placeholder for Spaces
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
  }
}

provider "digitalocean" {
  token = var.do_token
}

# ── Variables ──────────────────────────────────────────────────────────────

variable "do_token" {
  description = "DigitalOcean personal access token"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "Master password for kinara-prod-db"
  type        = string
  sensitive   = true
  default     = ""  # set via TF_VAR_db_password or .tfvars
}

variable "alert_email" {
  description = "Email for DigitalOcean alerts"
  type        = string
  default     = "donalddaglo@gmail.com"
}

# ── VPC ───────────────────────────────────────────────────────────────────

resource "digitalocean_vpc" "kinara_prod" {
  name     = "kinara-prod-vpc"
  region   = "fra1"
  ip_range = "10.10.0.0/16"
}

# ── Managed PostgreSQL cluster ────────────────────────────────────────────

resource "digitalocean_database_cluster" "kinara_prod_db" {
  name       = "kinara-prod-db"
  engine     = "pg"
  version    = "15"
  size       = "db-s-4vcpu-8gb"   # 4 vCPU / 8 GB RAM / 115 GB SSD (DO SKU)
  region     = "fra1"
  node_count = 2                   # primary + standby for HA

  private_network_uuid = digitalocean_vpc.kinara_prod.id

  maintenance_window {
    day  = "sunday"
    hour = "02:00:00"
  }

  tags = ["kinara-os", "production", "database"]
}

# Read-only replica for analytics queries (avoids OLAP load on primary)
resource "digitalocean_database_replica" "kinara_prod_db_replica" {
  cluster_id = digitalocean_database_cluster.kinara_prod_db.id
  name       = "kinara-prod-db-replica"
  size       = "db-s-2vcpu-4gb"
  region     = "fra1"

  private_network_uuid = digitalocean_vpc.kinara_prod.id

  tags = ["kinara-os", "production", "replica"]
}

# ── Firewall: restrict DB access to VPC only ────────────────────────────

resource "digitalocean_database_firewall" "kinara_db_fw" {
  cluster_id = digitalocean_database_cluster.kinara_prod_db.id

  rule {
    type  = "tag"
    value = "kinara-app"
  }

  # Allow Flyway migration runner (Bastion / CI runner) by tag
  rule {
    type  = "tag"
    value = "kinara-migrate"
  }
}

# ── Service databases (created inside the cluster) ────────────────────────
# DigitalOcean creates a default "defaultdb". We create one DB per service.

locals {
  service_databases = [
    "kinara_auth",
    "kinara_audit",
    "kinara_notification",
    "kinara_patient",
    "kinara_clinical",
    "kinara_appointment",
    "kinara_immunization",
    "kinara_lab",
    "kinara_outbreak",
    "kinara_pharmacy",
    "kinara_referral",
    "kinara_telemedicine",
    "kinara_health_analytics",
    "kinara_health_compliance",
    "kinara_farmer",
    "kinara_market",
    "kinara_cooperative",
    "kinara_weather",
    "kinara_input",
    "kinara_extension",
    "kinara_irrigation",
    "kinara_livestock",
    "kinara_farmer_finance",
    "kinara_fleet",
    "kinara_driver",
    "kinara_cargo",
    "kinara_route",
    "kinara_transport",
    "kinara_lastmile",
    "kinara_shipment",
    "kinara_logistics_analytics",
    "kinara_vehicle_tracking",
    "kinara_supply_chain",
    "kinara_warehouse",
    "kinara_port",
    "kinara_vessel",
    "kinara_dock",
    "kinara_cargo_maritime",
    "kinara_customs",
    "kinara_shipping",
    "kinara_crew",
    "kinara_voyage",
    "kinara_payment",
    "kinara_trade_finance",
    "kinara_documentation",
    "kinara_compliance",
    "kinara_analytics",
    "kinara_governance",
    "kinara_sms",
  ]
}

resource "digitalocean_database_db" "service_dbs" {
  for_each   = toset(local.service_databases)
  cluster_id = digitalocean_database_cluster.kinara_prod_db.id
  name       = each.key
}

# ── DB user (one shared app user; services connect with same creds) ───────

resource "digitalocean_database_user" "kinara_app" {
  cluster_id = digitalocean_database_cluster.kinara_prod_db.id
  name       = "kinara"
}

# ── Monitoring alert: CPU > 80% ───────────────────────────────────────────

resource "digitalocean_monitor_alert" "db_cpu" {
  alerts {
    email = [var.alert_email]
  }
  window      = "5m"
  type        = "v1/insights/droplet/cpu"
  compare     = "GreaterThan"
  value       = 80
  enabled     = true
  description = "Kinara prod DB CPU > 80%"
  tags        = ["kinara-os", "production"]
}

resource "digitalocean_monitor_alert" "db_disk" {
  alerts {
    email = [var.alert_email]
  }
  window      = "5m"
  type        = "v1/insights/droplet/disk_utilization_percent"
  compare     = "GreaterThan"
  value       = 85
  enabled     = true
  description = "Kinara prod DB disk > 85%"
  tags        = ["kinara-os", "production"]
}

# ── Outputs ───────────────────────────────────────────────────────────────

output "db_host" {
  value       = digitalocean_database_cluster.kinara_prod_db.private_host
  description = "Private host for app services (VPC access)"
}

output "db_public_host" {
  value       = digitalocean_database_cluster.kinara_prod_db.host
  description = "Public host for Flyway migration runner"
}

output "db_port" {
  value = digitalocean_database_cluster.kinara_prod_db.port
}

output "db_user" {
  value = digitalocean_database_user.kinara_app.name
}

output "db_password" {
  value     = digitalocean_database_user.kinara_app.password
  sensitive = true
}

output "db_uri" {
  value     = digitalocean_database_cluster.kinara_prod_db.private_uri
  sensitive = true
}

output "replica_host" {
  value       = digitalocean_database_replica.kinara_prod_db_replica.private_host
  description = "Analytics read replica private host"
}
