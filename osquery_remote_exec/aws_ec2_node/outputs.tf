output "instance_id" {
  description = "EC2 instance ID"
  value       = aws_instance.node.id
}

output "public_ip" {
  description = "Public IP address of the EC2 instance"
  value       = aws_instance.node.public_ip
}

output "private_ip" {
  description = "Private IP address of the EC2 instance"
  value       = aws_instance.node.private_ip
}

output "os" {
  description = "Operating system deployed"
  value       = var.os
}

output "ssh_user" {
  description = "Default SSH user for the selected OS"
  value = {
    ubuntu       = "ubuntu"
    debian       = "admin"
    amazon-linux = "ec2-user"
  }[var.os]
}

output "ssh_connection_string" {
  description = "SSH command to connect to the instance"
  value       = "ssh -i <your-key.pem> ${local.ssh_users[var.os]}@${aws_instance.node.public_ip}"

  depends_on = [aws_instance.node]
}

output "ssh_tunnel_command" {
  description = "SSH command to forward the osquery socket locally"
  value       = "ssh -fNT -L /tmp/lazyos_remote.sock:/var/osquery/osquery.em -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 ${local.ssh_users[var.os]}@${aws_instance.node.public_ip} -i <your-key.pem>"

  depends_on = [aws_instance.node]
}
