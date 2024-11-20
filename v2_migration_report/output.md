# AWS SDKv2 Migration Progress

The table below outlines the progress of the `AWS SDK` migration as detailed in [#745](https://github.com/gruntwork-io/cloud-nuke/issues/745).
run `go generate ./...` to refresh this report.


| Resource Name                    | Migrated           |
|----------------------------------|--------------------|
| accessanalyzer                   | :white_check_mark: |
| acm                              | :white_check_mark: |
| acmpca                           | :white_check_mark: |
| ami                              | :white_check_mark: |
| apigateway                       | :white_check_mark: |
| apigatewayv2                     | :white_check_mark: |
| app-runner-service               | :white_check_mark: |
| asg                              | :white_check_mark: |
| backup-vault                     | :white_check_mark: |
| cloudtrail                       | :white_check_mark: |
| cloudwatch-alarm                 | :white_check_mark: |
| cloudwatch-dashboard             | :white_check_mark: |
| cloudwatch-loggroup              | :white_check_mark: |
| codedeploy-application           | :white_check_mark: |
| config-recorders                 | :white_check_mark: |
| config-rules                     | :white_check_mark: |
| data-sync-location               | :white_check_mark: |
| data-sync-task                   | :white_check_mark: |
| dynamodb                         | :white_check_mark: |
| ebs                              | :white_check_mark: |
| ec2                              |                    |
| ec2-dedicated-hosts              | :white_check_mark: |
| ec2-endpoint                     | :white_check_mark: |
| ec2-keypairs                     | :white_check_mark: |
| ec2-placement-groups             | :white_check_mark: |
| ec2-subnet                       |                    |
| ec2_dhcp_option                  | :white_check_mark: |
| ecr                              | :white_check_mark: |
| ecscluster                       | :white_check_mark: |
| ecsserv                          | :white_check_mark: |
| efs                              | :white_check_mark: |
| egress-only-internet-gateway     |                    |
| eip                              | :white_check_mark: |
| ekscluster                       | :white_check_mark: |
| elastic-beanstalk                | :white_check_mark: |
| elasticache                      | :white_check_mark: |
| elasticacheParameterGroups       | :white_check_mark: |
| elasticacheSubnetGroups          | :white_check_mark: |
| elasticcache-serverless          | :white_check_mark: |
| elb                              | :white_check_mark: |
| elbv2                            | :white_check_mark: |
| event-bridge                     | :white_check_mark: |
| event-bridge-archive             | :white_check_mark: |
| event-bridge-rule                | :white_check_mark: |
| event-bridge-schedule            | :white_check_mark: |
| event-bridge-schedule-group      | :white_check_mark: |
| grafana                          | :white_check_mark: |
| guardduty                        | :white_check_mark: |
| iam                              | :white_check_mark: |
| iam-group                        | :white_check_mark: |
| iam-policy                       | :white_check_mark: |
| iam-role                         | :white_check_mark: |
| iam-service-linked-role          | :white_check_mark: |
| internet-gateway                 |                    |
| ipam                             | :white_check_mark: |
| ipam-byoasn                      | :white_check_mark: |
| ipam-custom-allocation           | :white_check_mark: |
| ipam-pool                        | :white_check_mark: |
| ipam-resource-discovery          | :white_check_mark: |
| ipam-scope                       | :white_check_mark: |
| kinesis-firehose                 | :white_check_mark: |
| kinesis-stream                   | :white_check_mark: |
| kmscustomerkeys                  | :white_check_mark: |
| lambda                           | :white_check_mark: |
| lambda_layer                     | :white_check_mark: |
| lc                               |                    |
| lt                               |                    |
| macie-member                     | :white_check_mark: |
| managed-prometheus               | :white_check_mark: |
| msk-cluster                      |                    |
| nat-gateway                      |                    |
| network-acl                      |                    |
| network-firewall                 | :white_check_mark: |
| network-firewall-policy          | :white_check_mark: |
| network-firewall-resource-policy | :white_check_mark: |
| network-firewall-rule-group      | :white_check_mark: |
| network-firewall-tls-config      | :white_check_mark: |
| network-interface                |                    |
| oidcprovider                     |                    |
| opensearchdomain                 | :white_check_mark: |
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
| s3                               | :white_check_mark: |
| s3-ap                            | :white_check_mark: |
| s3-mrap                          | :white_check_mark: |
| s3-olap                          | :white_check_mark: |
| sagemaker-notebook-smni          |                    |
| secretsmanager                   |                    |
| security-group                   |                    |
| security-hub                     | :white_check_mark: |
| ses-configuration-set            | :white_check_mark: |
| ses-email-template               | :white_check_mark: |
| ses-identity                     | :white_check_mark: |
| ses-receipt-filter               | :white_check_mark: |
| ses-receipt-rule-set             | :white_check_mark: |
| snap                             | :white_check_mark: |
| snstopic                         | :white_check_mark: |
| sqs                              | :white_check_mark: |
| transit-gateway                  | :white_check_mark: |
| transit-gateway-attachment       | :white_check_mark: |
| transit-gateway-route-table      |                    |
| vpc                              |                    |
| vpc-lattice-service              | :white_check_mark: |
| vpc-lattice-service-network      | :white_check_mark: |
| vpc-lattice-target-group         | :white_check_mark: |
