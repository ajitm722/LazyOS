package osquery

import "github.com/ajitm722/LazyOS/internal/daemons"

// CoreTables contains the prefilled catalog of the most useful tables for backend/system observability.
var CoreTables = []daemons.TableSchema{
	// Process & Memory
	{Name: "processes", Description: "All running processes on the host system.", Columns: "pid, name, path, cmdline, state, cwd, root, uid, gid, on_disk, resident_size, total_size"},
	{Name: "process_envs", Description: "Environment variables for running processes.", Columns: "pid, key, value"},
	{Name: "process_open_files", Description: "File descriptors for each process.", Columns: "pid, fd, path"},
	{Name: "process_open_sockets", Description: "Network sockets owned by a specific process.", Columns: "pid, fd, socket, family, protocol, local_address, remote_address, local_port, remote_port"},

	// Network
	{Name: "listening_ports", Description: "Ports currently listening on the host.", Columns: "pid, port, protocol, family, address, fd, socket"},
	{Name: "routes", Description: "The active route table for the host system.", Columns: "destination, netmask, gateway, source, flags, interface, metric"},
	{Name: "interface_addresses", Description: "Network interfaces and their addresses.", Columns: "interface, address, mask, broadcast, point_to_point, type"},
	{Name: "arp_cache", Description: "Address resolution cache.", Columns: "address, mac, interface"},
	{Name: "dns_resolvers", Description: "DNS resolvers configured for the host.", Columns: "id, type, address, netmask, options"},

	// Kernel & Hardware
	{Name: "kernel_info", Description: "Basic active kernel information.", Columns: "version, arguments, path, device"},
	{Name: "kernel_modules", Description: "Linux kernel modules both loaded and within the load search path.", Columns: "name, size, used_by, status, address"},
	{Name: "system_info", Description: "System hardware and software info.", Columns: "hostname, uuid, cpu_type, cpu_subtype, cpu_brand, cpu_physical_cores, cpu_logical_cores, physical_memory, hardware_vendor, hardware_model"},
	{Name: "uptime", Description: "System uptime in seconds.", Columns: "days, hours, minutes, seconds, total_seconds"},
	{Name: "mounts", Description: "System mounted file systems.", Columns: "device, device_alias, path, type, blocks_size, blocks, blocks_free, blocks_available, inodes, inodes_free, flags"},

	// Containers
	{Name: "docker_containers", Description: "Docker containers (requires Docker socket).", Columns: "id, name, image, image_id, command, created, state, status, pid, path"},
	{Name: "docker_images", Description: "Docker images on the host.", Columns: "id, tags, created"},
	{Name: "docker_networks", Description: "Docker networks on the host.", Columns: "id, name, driver, created, enable_ipv6, subnet, gateway"},

	// Users & Auth
	{Name: "users", Description: "Local system users.", Columns: "uid, gid, uid_signed, gid_signed, username, description, directory, shell, uuid"},
	{Name: "groups", Description: "Local system groups.", Columns: "gid, gid_signed, groupname"},
	{Name: "logged_in_users", Description: "Users currently logged in.", Columns: "type, user, tty, host, time, pid"},
	{Name: "authorized_keys", Description: "A line-delimited authorized_keys table.", Columns: "uid, algorithm, key, key_file"},
	{Name: "sudoers", Description: "Rules for running commands as other users via sudo.", Columns: "header, rule_details"},

	// System & Packages
	{Name: "os_version", Description: "A single row containing the operating system name and version.", Columns: "name, version, major, minor, patch, build, platform, platform_like, codename, arch"},
	{Name: "memory_info", Description: "Main memory information in bytes.", Columns: "memory_total, memory_free, buffers, cached, swap_cached, active, inactive, swap_total, swap_free"},
	{Name: "cpu_time", Description: "System CPU time data.", Columns: "core, user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice"},
	{Name: "crontab", Description: "Parsed crontab contents.", Columns: "event, minute, hour, day_of_month, month, day_of_week, command, path"},
	{Name: "suid_bin", Description: "suid binaries in common locations.", Columns: "path, username, groupname, permissions"},
	{Name: "iptables", Description: "Linux IP packet filtering and NAT tool.", Columns: "filter_name, chain, policy, target, protocol, src_ip, src_mask, dst_ip, dst_mask, iniface, outiface, match"},
}
