package util

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	autoscaling "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	cloudformationtypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iam "github.com/aws/aws-sdk-go-v2/service/iam/types"
	networkfirewalltypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	sagemakertypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	secretsmanagertypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// ConvertTagsToMap converts a slice of any tag-like struct to a map[string]string.
// keyFn and valFn extract the key and value from each tag element.
func ConvertTagsToMap[T any](tags []T, keyFn func(T) string, valFn func(T) string) map[string]string {
	result := make(map[string]string, len(tags))
	for _, tag := range tags {
		result[keyFn(tag)] = valFn(tag)
	}
	return result
}

func ConvertS3TypesTagsToMap(tags []s3types.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t s3types.Tag) string { return aws.ToString(t.Key) },
		func(t s3types.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertTypesTagsToMap(tags []ec2types.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t ec2types.Tag) string { return aws.ToString(t.Key) },
		func(t ec2types.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertSecretsManagerTagsToMap(tags []secretsmanagertypes.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t secretsmanagertypes.Tag) string { return aws.ToString(t.Key) },
		func(t secretsmanagertypes.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertAutoScalingTagsToMap(tags []autoscaling.TagDescription) map[string]string {
	return ConvertTagsToMap(tags,
		func(t autoscaling.TagDescription) string { return aws.ToString(t.Key) },
		func(t autoscaling.TagDescription) string { return aws.ToString(t.Value) },
	)
}

func ConvertStringPtrTagsToMap(tags map[string]*string) map[string]string {
	tagMap := make(map[string]string, len(tags))
	for key, value := range tags {
		tagMap[key] = aws.ToString(value)
	}
	return tagMap
}

func ConvertIAMTagsToMap(tags []iam.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t iam.Tag) string { return aws.ToString(t.Key) },
		func(t iam.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertRDSTypeTagsToMap(tags []rdstypes.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t rdstypes.Tag) string { return aws.ToString(t.Key) },
		func(t rdstypes.Tag) string { return aws.ToString(t.Value) },
	)
}

func GetEC2ResourceNameTagValue(tags []ec2types.Tag) *string {
	tagMap := ConvertTypesTagsToMap(tags)
	if name, ok := tagMap["Name"]; ok {
		return &name
	}
	return nil
}

func ConvertNetworkFirewallTagsToMap(tags []networkfirewalltypes.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t networkfirewalltypes.Tag) string { return aws.ToString(t.Key) },
		func(t networkfirewalltypes.Tag) string { return aws.ToString(t.Value) },
	)
}

// ConvertSageMakerTagsToMap converts SageMaker tags to a map[string]string
func ConvertSageMakerTagsToMap(tags []sagemakertypes.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t sagemakertypes.Tag) string { return aws.ToString(t.Key) },
		func(t sagemakertypes.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertRoute53TagsToMap(tags []route53types.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t route53types.Tag) string { return aws.ToString(t.Key) },
		func(t route53types.Tag) string { return aws.ToString(t.Value) },
	)
}

func ConvertCloudFormationTagsToMap(tags []cloudformationtypes.Tag) map[string]string {
	return ConvertTagsToMap(tags,
		func(t cloudformationtypes.Tag) string { return aws.ToString(t.Key) },
		func(t cloudformationtypes.Tag) string { return aws.ToString(t.Value) },
	)
}
