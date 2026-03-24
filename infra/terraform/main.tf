terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_s3_bucket" "scanner_results" {
  bucket = "arc-hawk-scanner-results"
}

resource "aws_s3_bucket_public_access_block" "scanner_results" {
  bucket = aws_s3_bucket.scanner_results.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
