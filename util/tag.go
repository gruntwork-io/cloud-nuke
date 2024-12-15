package util

import (
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	networkfirewalltypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func ConvertS3TypesTagsToMap(tags []s3types.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertTypesTagsToMap(tags []ec2types.Tag) map[string]string {
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

func ConvertRDSTypeTagsToMap(tags []rdstypes.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func GetEC2ResourceNameTagValue(tags []ec2types.Tag) *string {
	tagMap := ConvertTypesTagsToMap(tags)
	if name, ok := tagMap["Name"]; ok {
		return &name
	}

	return nil
}

func ConvertNetworkFirewallTagsToMap(tags []networkfirewalltypes.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}
