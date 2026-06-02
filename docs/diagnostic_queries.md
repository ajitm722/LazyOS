# Diagnostic Queries

This document serves as an index of diagnostic and audit workflows built on osquery tables. Each use case links to a dedicated guide containing annotated SQL queries, column interpretation tables, and result-reading strategies.

---

## Available Workflows

| Use Case | Domain | Tables Used |
|---|---|---|
| [Virtual Memory Inspection](./virtual_memory.md) | Kernel | `virtual_memory_info`, `process_virtual_memory`, `processes` |
| [AWS Bedrock Audit](./aws_bedrock.md) | Cloud (AWS) | `aws_bedrock_agent`, `aws_bedrock_knowledge_base`, `aws_bedrock_agent_action_group` |

---

Additional workflow guides will be added as new diagnostic and compliance use cases are identified.
