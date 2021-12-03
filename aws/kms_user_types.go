package aws

type KMSUserKeys struct {
	Aliases []string
}

// ResourceName - the simple name of the aws resource
func (c KMSUserKeys) ResourceName() string {
	return "kms-cmk"
}

// ResourceIdentifiers - The IAM UserNames
func (r KMSUserKeys) ResourceIdentifiers() []string {
	return r.Aliases
}

// MaxBatchSize - Requests batch size
func (r KMSUserKeys) MaxBatchSize() int {
	return 100
}
