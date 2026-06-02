package aws

import "github.com/ajitm722/LazyOS/internal/daemons"

// AWSTables contains the prefilled catalog of AWS resource tables from cloudquery.
var AWSTables = []daemons.TableSchema{
	// ── EC2 ──────────────────────────────────────────────────────
	{
		Name: "aws_ec2_instance", Description: "EC2 instances",
		Columns: "account_id, region_code, instances_instance_id, instances_instance_type, instances_launch_time, instances_vpc_id, instances_private_ip_address, instances_state",
	},
	{
		Name: "aws_ec2_vpc", Description: "EC2 VPCs",
		Columns: "account_id, region_code, vpc_id, cidr_block, is_default, state, owner_id",
	},
	{
		Name: "aws_ec2_subnet", Description: "EC2 subnets",
		Columns: "account_id, region_code, subnet_id, vpc_id, cidr_block, availability_zone, available_ip_address_count, state",
	},
	{
		Name: "aws_ec2_security_group", Description: "EC2 security groups",
		Columns: "account_id, region_code, group_id, group_name, description, vpc_id, owner_id",
	},
	{
		Name: "aws_ec2_route_table", Description: "EC2 route tables",
		Columns: "account_id, region_code, route_table_id, vpc_id, routes, owner_id",
	},
	{
		Name: "aws_ec2_network_acl", Description: "EC2 network ACLs",
		Columns: "account_id, region_code, network_acl_id, vpc_id, is_default, owner_id",
	},
	{
		Name: "aws_ec2_internet_gateway", Description: "EC2 internet gateways",
		Columns: "account_id, region_code, internet_gateway_id, attachments, owner_id",
	},
	{
		Name: "aws_ec2_keypair", Description: "EC2 key pairs",
		Columns: "account_id, region_code, key_name, key_pair_id, key_fingerprint",
	},
	{
		Name: "aws_ec2_tag", Description: "EC2 resource tags",
		Columns: "account_id, region_code, key, value, resource_id, resource_type",
	},

	// ── S3 ───────────────────────────────────────────────────────
	{
		Name: "aws_s3_bucket", Description: "S3 buckets",
		Columns: "account_id, region_code, name, creation_time, server_side_encryption_configuration, versioning_status, public_access_block_config",
	},

	// ── IAM ──────────────────────────────────────────────────────
	{
		Name: "aws_iam_user", Description: "IAM users",
		Columns: "account_id, user_id, user_name, arn, create_date, password_last_used",
	},
	{
		Name: "aws_iam_role", Description: "IAM roles",
		Columns: "account_id, role_id, role_name, arn, create_date, description, assume_role_policy_document",
	},
	{
		Name: "aws_iam_policy", Description: "IAM policies",
		Columns: "account_id, policy_id, policy_name, arn, create_date, attachment_count, is_attachable",
	},

	// ── RDS ──────────────────────────────────────────────────────
	{
		Name: "aws_rds_instance", Description: "RDS DB instances",
		Columns: "account_id, db_instance_identifier, db_instance_class, engine, engine_version, db_instance_status, multi_az, storage_encrypted",
	},

	// ── ELB ──────────────────────────────────────────────────────
	{
		Name: "aws_elbv2_loadbalancer", Description: "ALB/NLB load balancers",
		Columns: "account_id, region_code, load_balancer_name, dns_name, scheme, vpc_id, created_time",
	},

	// ── Containers ───────────────────────────────────────────────
	{
		Name: "aws_ecs_cluster", Description: "ECS clusters",
		Columns: "account_id, region_code, cluster_arns",
	},
	{
		Name: "aws_eks_cluster", Description: "EKS clusters",
		Columns: "account_id, region_code, clusters",
	},

	// ── Networking ───────────────────────────────────────────────
	{
		Name: "aws_sns_topic", Description: "SNS topics",
		Columns: "account_id, region_code, topic",
	},
	{
		Name: "aws_sqs_queue", Description: "SQS queues",
		Columns: "account_id, region_code, queue_urls",
	},

	// ── Security ─────────────────────────────────────────────────
	{
		Name: "aws_cloudtrail_trail", Description: "CloudTrail trails",
		Columns: "account_id, region_code, name, s3_bucket_name, is_multi_region_trail, is_organization_trail, log_file_validation_enabled, home_region",
	},

	// ── Bedrock ───────────────────────────────────────────────────
	{
		Name: "aws_bedrock_agent", Description: "Bedrock agents",
		Columns: "account_id, region_code, agent_id, agent_name, agent_arn, agent_status, description, latest_agent_version, updated_at",
	},
	{
		Name: "aws_bedrock_knowledge_base", Description: "Bedrock knowledge bases",
		Columns: "account_id, region_code, knowledge_base_id, name, description, status, updated_at",
	},
	{
		Name: "aws_bedrock_agent_action_group", Description: "Bedrock agent action groups",
		Columns: "account_id, region_code, action_group_id, name, state, description, updated_at",
	},
}
