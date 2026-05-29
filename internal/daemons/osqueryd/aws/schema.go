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
		Name: "aws_ec2_volume", Description: "EC2 EBS volumes",
		Columns: "account_id, region_code, volume_id, volume_type, size, state, encrypted, iops",
	},
	{
		Name: "aws_ec2_snapshot", Description: "EC2 EBS snapshots",
		Columns: "account_id, region_code, snapshot_id, volume_id, volume_size, state, encrypted, start_time",
	},
	{
		Name: "aws_ec2_keypair", Description: "EC2 key pairs",
		Columns: "account_id, region_code, key_name, key_pair_id, key_fingerprint",
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
		Name: "aws_ec2_nat_gateway", Description: "EC2 NAT gateways",
		Columns: "account_id, region_code, nat_gateway_id, subnet_id, vpc_id, state, create_time",
	},
	{
		Name: "aws_ec2_internet_gateway", Description: "EC2 internet gateways",
		Columns: "account_id, region_code, internet_gateway_id, attachments, owner_id",
	},
	{
		Name: "aws_ec2_egress_only_internet_gateway", Description: "EC2 egress-only IGWs (IPv6)",
		Columns: "account_id, region_code, egress_only_internet_gateway_id, attachments",
	},
	{
		Name: "aws_ec2_image", Description: "EC2 AMIs",
		Columns: "account_id, region_code, image_id, name, state, architecture, platform, creation_date",
	},
	{
		Name: "aws_ec2_flowlog", Description: "EC2 VPC flow logs",
		Columns: "account_id, region_code, flow_log_id, resource_id, traffic_type, log_destination, flow_log_status",
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
	{
		Name: "aws_iam_group", Description: "IAM groups",
		Columns: "account_id, group_id, group_name, arn, create_date",
	},
	{
		Name: "aws_iam_account_password_policy", Description: "IAM password policy",
		Columns: "account_id, allow_users_to_change_password, max_password_age, minimum_password_length, require_lowercase_characters, require_numbers, require_symbols, require_uppercase_characters",
	},

	// ── RDS ──────────────────────────────────────────────────────
	{
		Name: "aws_rds_instance", Description: "RDS DB instances",
		Columns: "account_id, db_instance_identifier, db_instance_class, engine, engine_version, db_instance_status, multi_az, storage_encrypted",
	},
	{
		Name: "aws_rds_cluster", Description: "RDS DB clusters",
		Columns: "account_id, db_cluster_identifier, engine, engine_mode, engine_version, status, multi_az, storage_encrypted",
	},
	{
		Name: "aws_rds_snapshot", Description: "RDS cluster snapshots",
		Columns: "account_id, DBClusterIdentifier, DBClusterSnapshotIdentifier, Engine, SnapshotType, Status, SnapshotCreateTime",
	},

	// ── ELB / ALB / NLB ─────────────────────────────────────────
	{
		Name: "aws_elbv2_loadbalancer", Description: "ALB/NLB load balancers",
		Columns: "account_id, region_code, load_balancer_name, dns_name, scheme, vpc_id, created_time",
	},
	{
		Name: "aws_elb_loadbalancer", Description: "Classic load balancers",
		Columns: "account_id, region_code, load_balancer_name, dns_name, scheme, vpc_id, created_time",
	},

	// ── Containers ───────────────────────────────────────────────
	{
		Name: "aws_ecr_repository", Description: "ECR repositories",
		Columns: "account_id, region_code, repository_name, repository_arn, repository_uri, created_at",
	},
	{
		Name: "aws_ecs_cluster", Description: "ECS clusters",
		Columns: "account_id, region_code, cluster_arns",
	},
	{
		Name: "aws_eks_cluster", Description: "EKS clusters",
		Columns: "account_id, region_code, clusters",
	},

	// ── Storage ──────────────────────────────────────────────────
	{
		Name: "aws_efs_file_system", Description: "EFS file systems",
		Columns: "account_id, region_code, file_system_id, name, life_cycle_state, encrypted, performance_mode, throughput_mode",
	},
	{
		Name: "aws_s3_glacier_vault", Description: "S3 Glacier vaults",
		Columns: "account_id, region_code, vault_name, vault_arn, creation_date, number_of_archives, size_in_bytes",
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

	// ── Security & Compliance ────────────────────────────────────
	{
		Name: "aws_kms_key", Description: "KMS keys",
		Columns: "account_id, region_code, key_arn, key_id",
	},
	{
		Name: "aws_cloudtrail_trail", Description: "CloudTrail trails",
		Columns: "account_id, region_code, name, s3_bucket_name, is_multi_region_trail, is_organization_trail, log_file_validation_enabled, home_region",
	},
	{
		Name: "aws_guardduty_detector", Description: "GuardDuty detectors",
		Columns: "account_id, region_code, detector_ids",
	},
	{
		Name: "aws_config_recorder", Description: "Config recorders",
		Columns: "account_id, region_code, name, role_arn, recording_group_all_supported, recording_group_include_global_resource_types",
	},
	{
		Name: "aws_config_delivery_channel", Description: "Config delivery channels",
		Columns: "account_id, region_code, name, s3_bucket_name, sns_topic_arn",
	},

	// ── Monitoring ───────────────────────────────────────────────
	{
		Name: "aws_cloudwatch_alarm", Description: "CloudWatch alarms",
		Columns: "account_id, region_code, alarm_name, alarm_arn, namespace, metric_name, state_value, threshold",
	},
	{
		Name: "aws_cloudwatch_event_rule", Description: "EventBridge rules",
		Columns: "account_id, region_code, name, arn, event_bus_name, schedule_expression, state",
	},
	{
		Name: "aws_cloudwatch_event_bus", Description: "EventBridge event buses",
		Columns: "account_id, region_code, name, arn, policy",
	},

	// ── Other Services ───────────────────────────────────────────
	{
		Name: "aws_acm_certificate", Description: "ACM certificates",
		Columns: "account_id, region_code, certificate_arn, domain_name",
	},
	{
		Name: "aws_cloudformation_stack", Description: "CloudFormation stacks",
		Columns: "account_id, region_code, stack_id, stack_name, stack_status, creation_time, last_updated_time, disable_rollback",
	},
	{
		Name: "aws_organizations_organization", Description: "Organizations master",
		Columns: "account_id, id, arn, feature_set, master_account_id, master_account_email",
	},
	{
		Name: "aws_organizations_account", Description: "Organizations accounts",
		Columns: "account_id, id, arn, name, email, status, joined_method",
	},
	{
		Name: "aws_organizations_root", Description: "Organizations roots",
		Columns: "account_id, id, arn, name, policy_types",
	},
	{
		Name: "aws_organizations_delegated_administrator", Description: "Organizations delegated admins",
		Columns: "account_id, id, arn, email, name, status, delegation_enabled_date",
	},
	{
		Name: "aws_apigateway_rest_api", Description: "API Gateway REST APIs",
		Columns: "account_id, region_code, id, name, description, created_date, endpoint_configuration",
	},
	{
		Name: "aws_workspaces_workspace", Description: "WorkSpaces",
		Columns: "account_id, region_code, workspace_id, directory_id, user_name, bundle_id, state, ip_address",
	},
	{
		Name: "aws_codepipeline_pipeline", Description: "CodePipeline pipelines",
		Columns: "account_id, region_code, name, version, created, updated",
	},
	{
		Name: "aws_codecommit_repository", Description: "CodeCommit repositories",
		Columns: "account_id, region_code, repository_id, repository_name",
	},
	{
		Name: "aws_codedeploy_application", Description: "CodeDeploy applications",
		Columns: "account_id, region_code, applications",
	},
	{
		Name: "aws_directoryservice_directory", Description: "Directory Service directories",
		Columns: "account_id, region_code, directory_id, name, type, size, stage, edition",
	},
}
