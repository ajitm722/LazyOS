provider "aws" {
  region = var.region
}

locals {
  os_config = {
    ubuntu = {
      owners     = ["099720109477"]
      name_filter = "ubuntu/images/hvm-ssd/ubuntu-*-*-amd64-server-*"
    }
    debian = {
      owners     = ["136693071363"]
      name_filter = "debian-*-amd64-*"
    }
    amazon-linux = {
      owners     = ["137112412989"]
      name_filter = "amzn2-ami-hvm-*-x86_64-gp2"
    }
  }

  selected_os  = local.os_config[var.os]
  default_tags = {
    Name    = "${var.name_prefix}-node"
    Managed = "opentofu"
  }
}

data "aws_ami" "selected" {
  most_recent = true
  owners      = local.selected_os.owners

  filter {
    name   = "name"
    values = [local.selected_os.name_filter]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_security_group" "node" {
  name_prefix = "${var.name_prefix}-sg-"
  description = "Security group for LazyOS remote node"
  tags        = local.default_tags

  ingress {
    description = "SSH from allowed CIDR blocks"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "node" {
  ami                    = data.aws_ami.selected.id
  instance_type          = var.instance_type
  key_name               = var.ssh_key_name
  vpc_security_group_ids = [aws_security_group.node.id]

  associate_public_ip_address = true

  tags = local.default_tags
}
