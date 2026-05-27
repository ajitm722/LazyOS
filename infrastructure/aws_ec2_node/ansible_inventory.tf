locals {
  inventory_content = templatefile("${path.module}/templates/ansible_inventory.tftpl", {
    public_ip  = aws_instance.node.public_ip
    ssh_user   = local.ssh_users[var.os]
    os         = var.os
    instance_id = aws_instance.node.id
  })
}

resource "local_file" "ansible_inventory" {
  filename        = "${path.module}/../ansible/inventory.ini"
  content         = local.inventory_content
  file_permission = "0644"

  depends_on = [aws_instance.node]
}
