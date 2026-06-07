terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.16.0"
    }
  }

  required_version = ">= 1.13"

  backend "s3" {
    bucket       = "wind-alert-terraform-state-513532022998"
    key          = "wind-alert/terraform.tfstate"
    region       = "eu-central-1"
    encrypt      = true
    use_lockfile = true
  }
}

