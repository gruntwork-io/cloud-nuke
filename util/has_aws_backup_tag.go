package util

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Check if the image has an AWS Backup tag
// Resources created by AWS Backup are listed as owned by self, but are actually
// AWS managed resources and cannot be deleted here.
func HasAWSBackupTag(tags []*ec2.Tag) bool {
	t := make(map[string]string)

	for _, v := range tags {
		t[awsgo.StringValue(v.Key)] = awsgo.StringValue(v.Value)
	}

	if _, ok := t["aws:backup:source-resource"]; ok {
		return true
	}
	return false
}
