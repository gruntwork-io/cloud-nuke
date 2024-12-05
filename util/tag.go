package util

import (
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/networkfirewall"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
)

func ConvertS3TagsToMap(tags []*s3.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertEC2TagsToMap(tags []*ec2.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertTypesTagsToMap(tags []types.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertAutoScalingTagsToMap(tags []autoscaling.TagDescription) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertStringPtrTagsToMap(tags map[string]*string) map[string]string {
	tagMap := make(map[string]string)
	for key, value := range tags {
		tagMap[key] = *value
	}

	return tagMap
}

func ConvertIAMTagsToMap(tags []iam.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertRDSTagsToMap(tags []*rds.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func GetEC2ResourceNameTagValue[T *ec2.Tag | types.Tag](tags []T) *string {
	var tagMap map[string]string

	switch t := any(tags).(type) {
	case []*ec2.Tag:
		tagMap = ConvertEC2TagsToMap(t)
	case []types.Tag:
		tagMap = ConvertTypesTagsToMap(t)
	default:
		return nil
	}

	if name, ok := tagMap["Name"]; ok {
		return &name
	}
	return nil
}

func ConvertNetworkFirewallTagsToMap(tags []*networkfirewall.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}
