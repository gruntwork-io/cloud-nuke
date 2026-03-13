package util

import (
	"testing"

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
	"github.com/stretchr/testify/assert"
)

func sp(s string) *string { return &s }

func TestConvertS3TypesTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertS3TypesTagsToMap(nil))
	assert.Equal(t, map[string]string{"k": "v"},
		ConvertS3TypesTagsToMap([]s3types.Tag{{Key: sp("k"), Value: sp("v")}}))
	// nil key/value skipped
	assert.Equal(t, map[string]string{"ok": "yes"},
		ConvertS3TypesTagsToMap([]s3types.Tag{{Key: nil, Value: sp("v")}, {Key: sp("ok"), Value: sp("yes")}}))
}

func TestConvertTypesTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertTypesTagsToMap(nil))
	assert.Equal(t, map[string]string{"Name": "i"},
		ConvertTypesTagsToMap([]ec2types.Tag{{Key: sp("Name"), Value: sp("i")}}))
	// nil key/value skipped
	assert.Empty(t, ConvertTypesTagsToMap([]ec2types.Tag{{Key: nil, Value: sp("v")}}))
}

func TestConvertSecretsManagerTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertSecretsManagerTagsToMap(nil))
	assert.Equal(t, map[string]string{"s": "v"},
		ConvertSecretsManagerTagsToMap([]secretsmanagertypes.Tag{{Key: sp("s"), Value: sp("v")}}))
}

func TestConvertAutoScalingTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertAutoScalingTagsToMap(nil))
	assert.Equal(t, map[string]string{"a": "b"},
		ConvertAutoScalingTagsToMap([]autoscaling.TagDescription{{Key: sp("a"), Value: sp("b")}}))
}

func TestConvertStringPtrTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertStringPtrTagsToMap(nil))
	assert.Equal(t, map[string]string{"k": "v"},
		ConvertStringPtrTagsToMap(map[string]*string{"k": sp("v")}))
}

func TestConvertIAMTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertIAMTagsToMap(nil))
	assert.Equal(t, map[string]string{"r": "admin"},
		ConvertIAMTagsToMap([]iam.Tag{{Key: sp("r"), Value: sp("admin")}}))
}

func TestConvertRDSTypeTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertRDSTypeTagsToMap(nil))
	assert.Equal(t, map[string]string{"db": "pg"},
		ConvertRDSTypeTagsToMap([]rdstypes.Tag{{Key: sp("db"), Value: sp("pg")}}))
}

func TestGetEC2ResourceNameTagValue(t *testing.T) {
	assert.Nil(t, GetEC2ResourceNameTagValue(nil))
	assert.Nil(t, GetEC2ResourceNameTagValue([]ec2types.Tag{{Key: sp("env"), Value: sp("prod")}}))
	result := GetEC2ResourceNameTagValue([]ec2types.Tag{{Key: sp("Name"), Value: sp("my-i")}})
	assert.Equal(t, "my-i", *result)
}

func TestConvertNetworkFirewallTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertNetworkFirewallTagsToMap(nil))
	assert.Equal(t, map[string]string{"fw": "m"},
		ConvertNetworkFirewallTagsToMap([]networkfirewalltypes.Tag{{Key: sp("fw"), Value: sp("m")}}))
}

func TestConvertSageMakerTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertSageMakerTagsToMap(nil))
	assert.Equal(t, map[string]string{"m": "v1"},
		ConvertSageMakerTagsToMap([]sagemakertypes.Tag{{Key: sp("m"), Value: sp("v1")}}))
	// nil key/value skipped
	assert.Empty(t, ConvertSageMakerTagsToMap([]sagemakertypes.Tag{{Key: nil, Value: sp("v")}}))
}

func TestConvertRoute53TagsToMap(t *testing.T) {
	assert.Empty(t, ConvertRoute53TagsToMap(nil))
	assert.Equal(t, map[string]string{"z": "ex.com"},
		ConvertRoute53TagsToMap([]route53types.Tag{{Key: sp("z"), Value: sp("ex.com")}}))
}

func TestConvertCloudFormationTagsToMap(t *testing.T) {
	assert.Empty(t, ConvertCloudFormationTagsToMap(nil))
	assert.Equal(t, map[string]string{"s": "main"},
		ConvertCloudFormationTagsToMap([]cloudformationtypes.Tag{{Key: sp("s"), Value: sp("main")}}))
}
