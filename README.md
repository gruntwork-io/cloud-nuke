aws-nuke
========

Delete all resources for a specific AWS account.

Usage
-----

    usage: aws-nuke [--non-interactive] id

    Delete all aws resources for the provided account id.

    Options:
        --non-interactive  Assume yes for all interactive prompts   

TODO
----

1.  Remove all non-protected EC2 instances
2.  Remove all elb's
3.  Remove all unattached EBS volumes
4.  Remove all unattached ENI's
5.  Remove all unused Security Groups
6.  Remove all unused Elastic IP's
7.  Remove all kinesis streams
8.  Remove all Auto Scale Groups (ASG)
9.  Remove all EC2 Container Service (ECS) clusters
10. Remove all EC2 Container Registry (ECR) repos
11. Remove all non-default Virtual Private Clouds (VPC's)
12. Remove all non-default subnets
13. Remove all non-default route tables
14. Remove all Simple Storage Service (S3) buckets
15. Remove all Simple Queue Service (SQS) queues
16. Remove all Lambda Functions
17. Remove all CloudFront distributions
18. Remove all ElasticCache Instances
19. Time based filtering
20. Region based filtering
21. Opt-Out specific aws resources from being nuked (ie EC2)

