package util

import (
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	networkfirewalltypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
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

// ConvertSageMakerTagsToMap converts SageMaker tags to a map[string]string
func ConvertSageMakerTagsToMap(tags []sagemakertypes.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

func ConvertRoute53TagsToMap(tags []route53types.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

func ConvertCloudFormationTagsToMap(tags []cloudformationtypes.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}
