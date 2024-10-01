# AWS SDKv2 Migration Progress

The table below outlines the progress of the `AWS SDK` migration as detailed in [#745](https://github.com/gruntwork-io/cloud-nuke/issues/745).
run `go generate ./...` to refresh this report.


| Resource Name                    | Migrated           |
|----------------------------------|--------------------|
| accessanalyzer                   |                    |
| acm                              |                    |
| acmpca                           |                    |
| ami                              |                    |
| apigateway                       |                    |
| apigatewayv2                     |                    |
| app-runner-service               |                    |
| asg                              |                    |
| backup-vault                     |                    |
| cloudtrail                       |                    |
| cloudwatch-alarm                 |                    |
| cloudwatch-dashboard             |                    |
| cloudwatch-loggroup              |                    |
| codedeploy-application           |                    |
| config-recorders                 |                    |
| config-rules                     |                    |
| data-sync-location               |                    |
| data-sync-task                   |                    |
| dynamodb                         |                    |
| ebs                              |                    |
| ec2                              |                    |
| ec2-dedicated-hosts              |                    |
| ec2-endpoint                     |                    |
| ec2-keypairs                     |                    |
| ec2-placement-groups             |                    |
| ec2-subnet                       |                    |
| ec2_dhcp_option                  |                    |
| ecr                              |                    |
| ecscluster                       |                    |
| ecsserv                          |                    |
| efs                              |                    |
| egress-only-internet-gateway     |                    |
| eip                              |                    |
| ekscluster                       |                    |
| elastic-beanstalk                |                    |
| elasticache                      |                    |
| elasticacheParameterGroups       |                    |
| elasticacheSubnetGroups          |                    |
| elb                              |                    |
| elbv2                            |                    |
| event-bridge                     | :white_check_mark: |
| event-bridge-archive             | :white_check_mark: |
| event-bridge-rule                | :white_check_mark: |
| event-bridge-schedule            | :white_check_mark: |
| event-bridge-schedule-group      | :white_check_mark: |
| guardduty                        |                    |
| iam                              |                    |
| iam-group                        |                    |
| iam-policy                       |                    |
| iam-role                         |                    |
| iam-service-linked-role          |                    |
| internet-gateway                 |                    |
| ipam                             |                    |
| ipam-byoasn                      |                    |
| ipam-custom-allocation           |                    |
| ipam-pool                        |                    |
| ipam-resource-discovery          |                    |
| ipam-scope                       |                    |
| kinesis-firehose                 |                    |
| kinesis-stream                   |                    |
| kmscustomerkeys                  |                    |
| lambda                           |                    |
| lambda_layer                     |                    |
| lc                               |                    |
| lt                               |                    |
| macie-member                     |                    |
| managed-prometheus               | :white_check_mark: |
| msk-cluster                      |                    |
| nat-gateway                      |                    |
| network-acl                      |                    |
| network-firewall                 |                    |
| network-firewall-policy          |                    |
| network-firewall-resource-policy |                    |
| network-firewall-rule-group      |                    |
| network-firewall-tls-config      |                    |
| network-interface                |                    |
| oidcprovider                     |                    |
| opensearchdomain                 |                    |
| rds                              |                    |
| rds-cluster                      |                    |
| rds-global-cluster               |                    |
| rds-global-cluster-membership    |                    |
| rds-parameter-group              |                    |
| rds-proxy                        |                    |
| rds-snapshot                     |                    |
| rds-subnet-group                 |                    |
| redshift                         |                    |
| route53-cidr-collection          |                    |
| route53-hosted-zone              |                    |
| route53-traffic-policy           |                    |
| s3                               |                    |
| s3-ap                            |                    |
| s3-mrap                          |                    |
| s3-olap                          |                    |
| sagemaker-notebook-smni          |                    |
| secretsmanager                   |                    |
| security-group                   |                    |
| security-hub                     |                    |
| ses-configuration-set            |                    |
| ses-email-template               |                    |
| ses-identity                     |                    |
| ses-receipt-filter               |                    |
| ses-receipt-rule-set             |                    |
| snap                             |                    |
| snstopic                         | :white_check_mark: |
| sqs                              | :white_check_mark: |
| transit-gateway                  |                    |
| transit-gateway-attachment       |                    |
| transit-gateway-route-table      |                    |
| vpc                              |                    |
| vpc-lattice-service              |                    |
| vpc-lattice-service-network      |                    |
| vpc-lattice-target-group         |                    |
