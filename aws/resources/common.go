package resources

// The pattern we use for running the `cloud-nuke` tool is to split the AWS API calls
// into batches when the function `NukeAllResources` is executed.
// A batch max number has been chosen for most modules.
// However, for ECS clusters there is no explicit limiting described in the AWS CLI docs.
// Therefore this `maxBatchSize` here is set to 49 as a safe maximum.

// This constant was moved into `common.go` since it was referenced across multiple resource types
const maxBatchSize = 49
