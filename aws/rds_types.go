package aws

type RdsDeleteError struct{}

func (e RdsDeleteError) Error() string {
	return "RDS DB Instance was not deleted"
}
