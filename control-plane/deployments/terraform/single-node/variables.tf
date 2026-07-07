variable "aws_region" {
  description = "AWS Region to deploy to"
  type        = string
  default     = "eu-north-1"
}

variable "instance_type" {
  description = "EC2 Instance type (Free Tier eligible)"
  type        = string
  default     = "t3.micro"
}
