package aws

// import (
// 	awsgo "github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/aws/session"
// 	"github.com/gruntwork-io/go-commons/errors"
// )

// EFSInstances - represents all EFS instances
type EFSInstances struct {
	FileSystemIds []string
}

// ResourceName - the simple name of the aws resource
func (fileSystem EFSInstances) ResourceName() string {
	return "efs"
}

// ResourceIdentifiers - The instance ids of the EFS instances
func (fileSystem EFSInstances) ResourceIdentifiers() []string {
	return fileSystem.FileSystemIds
}

func (fileSystem EFSInstances) MaxBatchSize() int {
	// Tentative batch size to ensure AWS doesn't throttle
	return 10
}
