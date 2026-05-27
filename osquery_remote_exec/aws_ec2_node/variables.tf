variable "region" {
  description = "AWS region to deploy the EC2 instance into"
  type        = string
  default     = "us-east-1"
}

variable "instance_type" {
  description = "EC2 instance type (t3.micro is Free Tier eligible)"
  type        = string
  default     = "t3.micro"
}

variable "os" {
  description = "Operating system for the EC2 instance"
  type        = string
  default     = "ubuntu"

  validation {
    condition     = contains(["ubuntu", "debian", "amazon-linux"], var.os)
    error_message = "OS must be one of: ubuntu, debian, amazon-linux"
  }
}

variable "ssh_key_name" {
  description = "Name of the AWS key pair to associate with the instance"
  type        = string
}

variable "ssh_cidr_blocks" {
  description = "CIDR blocks allowed to SSH into the instance. WARNING: default 0.0.0.0/0 opens SSH to the entire internet — lock to your IP before production use."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
  default     = "lazyos"
}
