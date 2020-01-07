package aws

type RdsDeleteError struct{}

func (e RdsDeleteError) Error() string {
  return "RDS was not deleted"
}

