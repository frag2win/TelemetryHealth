variable "aws_region" {
  description = "AWS Region to deploy to"
  type        = string
  default     = "eu-north-1"
}

variable "instance_type" {
  description = "EC2 Instance type (must have at least 4GB RAM)"
  type        = string
  default     = "t3.medium"
}
