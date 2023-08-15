package commands

import (
	"testing"
	"time"

	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	now := time.Now()
	then, err := parseDurationParam("1h")
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if now.Hour() == 0 {
		// At midnight, now.Hour returns 0 so we need to handle that specially.
		assert.Equal(t, 23, then.Hour())
		// Also, the date changed, so 1 hour ago will be the previous day.
		assert.Equal(t, now.Day()-1, then.Day())
	} else {
		assert.Equal(t, now.Hour()-1, then.Hour())
		assert.Equal(t, now.Day(), then.Day())
	}

	assert.Equal(t, now.Month(), then.Month())
	assert.Equal(t, now.Year(), then.Year())
}

func TestParseDurationInvalidFormat(t *testing.T) {
	_, err := parseDurationParam("")
	assert.Error(t, err)
}

func TestListResourceTypes(t *testing.T) {
	allAWSResourceTypes := aws.ListResourceTypes()
	assert.Greater(t, len(allAWSResourceTypes), 0)
	assert.Contains(t, allAWSResourceTypes, (&aws.EC2Instances{}).ResourceName())
}

func TestIsValidResourceType(t *testing.T) {
	allAWSResourceTypes := aws.ListResourceTypes()
	ec2ResourceName := (*&aws.EC2Instances{}).ResourceName()
	assert.Equal(t, aws.IsValidResourceType(ec2ResourceName, allAWSResourceTypes), true)
	assert.Equal(t, aws.IsValidResourceType("xyz", allAWSResourceTypes), false)
}

func TestIsNukeable(t *testing.T) {
	ec2ResourceName := (&aws.EC2Instances{}).ResourceName()
	amiResourceName := (&aws.AMIs{}).ResourceName()

	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{ec2ResourceName}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{"all"}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{}), true)
	assert.Equal(t, aws.IsNukeable(ec2ResourceName, []string{amiResourceName}), false)
}
