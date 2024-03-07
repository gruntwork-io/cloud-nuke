package config

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func emptyConfig() *Config {
	return &Config{
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		KMSCustomerKeyResourceType{false, ResourceType{FilterRule{}, FilterRule{}}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
		ResourceType{FilterRule{}, FilterRule{}},
	}
}

func TestConfig_Garbage(t *testing.T) {
	configFilePath := "./mocks/garbage.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestConfig_Malformed(t *testing.T) {
	configFilePath := "./mocks/malformed.yaml"
	_, err := GetConfig(configFilePath)

	// Expect malformed to throw a yaml TypeError
	require.Error(t, err, "Received expected error")
	return
}

func TestConfig_Empty(t *testing.T) {
	configFilePath := "./mocks/empty.yaml"
	configObj, err := GetConfig(configFilePath)

	require.NoError(t, err)

	if !reflect.DeepEqual(configObj, emptyConfig()) {
		assert.Fail(t, "Config should be empty, %+v\n", configObj)
	}

	return
}

func TestShouldInclude_AllowWhenEmpty(t *testing.T) {
	var includeREs []Expression
	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-open-vpn", includeREs, excludeREs),
		"Should include when both lists are empty")
}

func TestShouldInclude_ExcludeWhenMatches(t *testing.T) {
	var includeREs []Expression

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches from the 'exclude' list")
	assert.True(t, ShouldInclude("tf-state-bucket", includeREs, excludeREs),
		"Should include when doesn't matches from the 'exclude' list")
}

func TestShouldInclude_IncludeWhenMatches(t *testing.T) {
	include, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	var excludeREs []Expression

	assert.True(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should include when matches the 'include' list")
	assert.False(t, ShouldInclude("test-vpc-123", includeREs, excludeREs),
		"Should not include when doesn't matches the 'include' list")
}

func TestShouldInclude_WhenMatchesIncludeAndExclude(t *testing.T) {
	include, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	includeREs := []Expression{{RE: *include}}

	exclude, err := regexp.Compile(`.*openvpn.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	assert.True(t, ShouldInclude("test-eks-cluster-123", includeREs, excludeREs),
		"Should include when matches the 'include' list but not matches the 'exclude' list")
	assert.False(t, ShouldInclude("test-openvpn-123", includeREs, excludeREs),
		"Should not include when matches 'exclude' list")
	assert.False(t, ShouldInclude("terraform-tf-state", includeREs, excludeREs),
		"Should not include when doesn't matches 'include' list")
}

func TestShouldIncludeBasedOnTime_IncludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeBefore: &now},
	}
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_IncludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		IncludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeBefore(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeBefore: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
}

func TestShouldIncludeBasedOnTime_ExcludeTimeAfter(t *testing.T) {
	now := time.Now()

	r := ResourceType{
		ExcludeRule: FilterRule{TimeAfter: &now},
	}
	assert.False(t, r.ShouldIncludeBasedOnTime(now.Add(1)))
	assert.True(t, r.ShouldIncludeBasedOnTime(now.Add(-1)))
}

func TestShouldInclude_NameAndTimeFilter(t *testing.T) {
	now := time.Now()

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}
	r := ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   &now,
		},
	}

	// Filter by Time
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Filter by Name
	assert.False(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("test_hello_world"),
		Time: aws.Time(now.Add(1)),
	}))
	// Pass filters
	assert.True(t, r.ShouldInclude(ResourceValue{
		Name: aws.String("hello_world"),
		Time: aws.Time(now.Add(-1)),
	}))
}

func TestAddIncludeAndExcludeAfterTime(t *testing.T) {
	now := aws.Time(time.Now())

	exclude, err := regexp.Compile(`test.*`)
	require.NoError(t, err)
	excludeREs := []Expression{{RE: *exclude}}

	testConfig := &Config{}
	testConfig.ACM = ResourceType{
		ExcludeRule: FilterRule{
			NamesRegExp: excludeREs,
			TimeAfter:   now,
		},
	}

	testConfig.AddExcludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Nil(t, testConfig.ACM.IncludeRule.TimeAfter)

	testConfig.AddIncludeAfterTime(now)
	assert.Equal(t, testConfig.ACM.ExcludeRule.NamesRegExp, excludeREs)
	assert.Equal(t, testConfig.ACM.ExcludeRule.TimeAfter, now)
	assert.Equal(t, testConfig.ACM.IncludeRule.TimeAfter, now)
}
