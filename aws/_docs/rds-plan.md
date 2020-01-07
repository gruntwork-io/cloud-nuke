# RDS Plan

Hello!

Looking up to the existing tests it's possible to create something similar that creates and deletes the instances so I'll try to avoid using the UI. I'm thinking about following these steps:

1. Create a new RDS instance
1. The lookp for the instances based on region
1. Nuke the instance based on the lookup results (with SkipFinalSnapshot: true)

1. Add the RDS to be searchable by the `--resource-type` flag.
1. Get instances by age (older than xx)

Aurora has a different criteria for deletion, according to [AWS documentation](https://docs.aws.amazon.com/sdk-for-go/api/service/rds/#RDS.DeleteDBInstance), isn't possible to delete using the same API as the others if it matches one of the following conditions:

```
* The DB cluster is a Read Replica of another Amazon Aurora DB cluster.

* The DB instance is the only instance in the DB cluster.
```

So after the non-Aurora part is completed, I'll

1. Create a new RDS Aurora Cluster
  - with only one instance
  - with a read replica

1. Lookup for the instances (probably it's the same API but not sure)
1. Nuke the Aurora instances

