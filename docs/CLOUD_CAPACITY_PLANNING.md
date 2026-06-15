# SignalPlane Silver Cloud Capacity Planning

Use this document to estimate the cost of running a production-grade Silver environment in OCI, AWS, or Google Cloud. It mirrors the on-prem baseline so the same product architecture can move between customer data centers and cloud.

## Estimate Assumptions

- One production environment.
- Three availability zones or fault domains where available.
- 50-100 monitored services.
- 500-1,000 hosts or pods.
- 100 GB/day raw logs, metrics, and traces combined.
- 30 days hot retention in ClickHouse.
- 180 days archive retention in object storage.
- HA for SignalPlane, PostgreSQL, ClickHouse, ingress, and collectors.

Also ask the cloud team to price a growth case at 250 GB/day raw telemetry.

## Common Bill Of Materials

| Area | Capacity |
|---|---|
| Kubernetes | 1 regional or multi-AZ cluster |
| SignalPlane node pool | 3 nodes, 4 vCPU and 16 GB RAM each, autoscale 3-6 |
| Collector node pool | 3 nodes, 4 vCPU and 16 GB RAM each, autoscale 3-6 |
| ClickHouse node pool | 3 nodes, 16 vCPU and 64-128 GB RAM each |
| ClickHouse storage | 2 TB fast SSD/block storage per node |
| PostgreSQL | HA, 4 vCPU, 16 GB RAM, 250 GB SSD, PITR |
| Object storage | 10 TB initial archive/backup storage |
| Registry | 100 GB private registry storage |
| Load balancer | 1 internal LB, optional public LB |
| Security | KMS, secrets, TLS certificates, private networking |
| Operations | Cloud monitoring/logging or customer Prometheus/Grafana |

## OCI Mapping

- Kubernetes: OCI Container Engine for Kubernetes.
- App/collector workers: VM.Standard.E5.Flex or equivalent, 3 nodes of 4 vCPU/16 GB each. In OCI calculator terms this is typically 2 OCPU/16 GB per node.
- ClickHouse workers: VM.Standard.E5.Flex or storage-optimized equivalent, 3 nodes of 16 vCPU/64-128 GB each. In OCI calculator terms start with 8 OCPU/64-128 GB per node.
- PostgreSQL: OCI Database with PostgreSQL, HA, 4 vCPU/16 GB, 250 GB.
- Storage: OCI Block Volume balanced or higher performance for ClickHouse, 2 TB per node.
- Archive: OCI Object Storage, 10 TB.
- Other: OCI Load Balancer, OCI Container Registry, OCI Vault/KMS, OCI DNS, OCI Monitoring/Logging, optional WAF.
- Estimator: https://www.oracle.com/cloud/costestimator.html

## AWS Mapping

- Kubernetes: Amazon EKS, multi-AZ.
- App/collector workers: EC2 m7i.xlarge or equivalent, 3 app nodes and 3 collector nodes.
- ClickHouse workers: EC2 m7i.4xlarge for 16 vCPU/64 GiB or r7i.4xlarge for 16 vCPU/128 GiB.
- PostgreSQL: Amazon RDS for PostgreSQL Multi-AZ, 4 vCPU/16 GiB class, 250 GB, backups/PITR.
- Storage: EBS gp3 or io2, 2 TB per ClickHouse node.
- Archive: S3, 10 TB.
- Other: ALB/NLB, ECR, Secrets Manager, KMS, Route 53, ACM, CloudWatch, optional WAF, optional SES or approved SMTP relay.
- Estimator: https://calculator.aws/

## Google Cloud Mapping

- Kubernetes: GKE Standard regional cluster.
- App/collector workers: Compute Engine n2-standard-4 or equivalent, 3 app nodes and 3 collector nodes.
- ClickHouse workers: n2-standard-16 for 16 vCPU/64 GB or n2-highmem-16 for more memory.
- PostgreSQL: Cloud SQL for PostgreSQL HA, 4 vCPU/16 GB, 250 GB SSD, backups/PITR.
- Storage: Hyperdisk Balanced or Persistent Disk SSD, 2 TB per ClickHouse node.
- Archive: Cloud Storage, 10 TB.
- Other: Cloud Load Balancing, Artifact Registry, Secret Manager, Cloud KMS, Cloud DNS, Certificate Manager, Cloud Monitoring/Logging, approved SMTP provider.
- Estimator: https://cloud.google.com/products/calculator

## Cost Drivers

- Telemetry volume and retention dominate cost.
- Cross-zone replication and data transfer can materially affect ClickHouse cost.
- Public ingress, WAF, NAT gateway, and internet egress should be estimated explicitly.
- Managed PostgreSQL is preferred; self-managed PostgreSQL is only for customers that cannot use a managed DB.
- Storage class performance matters more than raw capacity for ClickHouse ingestion.
