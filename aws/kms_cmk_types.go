package aws

type KMSCMKeys struct {
	Aliases []string
}

// ResourceName - the simple name of the aws resource
func (c KMSCMKeys) ResourceName() string {
	return "kms-cmk"
}

// ResourceIdentifiers - The IAM UserNames
func (r KMSCMKeys) ResourceIdentifiers() []string {
	return r.Aliases
}

//
func (r KMSCMKeys) MaxBatchSize() int {
	return 200
}