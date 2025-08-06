terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = " =6.7.0"
    }

    local = {
      source  = "hashicorp/local"
      version = " = 2.5.3"
    }
  }

  required_version = "= 1.12.2"

  # tfstateはS3で管理する
  backend "s3" {
    bucket       = "management-enemaru-terraform-state"
    key          = "data-controller/terraform.tfstate"
    region       = "us-east-1"
    encrypt      = true
    use_lockfile = true
  }
}

provider "aws" {
  region = "us-east-1"
}
