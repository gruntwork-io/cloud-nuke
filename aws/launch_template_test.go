package aws

import (
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/go-commons/errors"
	"github.com/stretchr/testify/assert"
)

func createTestLaunchTemplate(t *testing.T, session *session.Session, name string) {
	svc := ec2.New(session)

	param := &ec2.CreateLaunchTemplateInput{
		LaunchTemplateName: &name,
		LaunchTemplateData: &ec2.RequestLaunchTemplateData{
			InstanceType: awsgo.String("t2.micro"),
		},
		VersionDescription: aws.String("cloud-nuke-test-v1"),
	}

	_, err := svc.CreateLaunchTemplate(param)
	if err != nil {
		assert.Failf(t, "Could not create test Launch template", errors.WithStackTrace(err).Error())
	}

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}
}

func TestListLaunchTemplates(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	createTestLaunchTemplate(t, session, uniqueTestID)

	// clean up after this test
	defer nukeAllLaunchTemplates(session, []*string{&uniqueTestID})

	templateNames, err := getAllLaunchTemplates(session, region, time.Now().Add(1*time.Hour*-1), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Templates")
	}

	// Template should not be in the list due to the time filter
	assert.NotContains(t, awsgo.StringValueSlice(templateNames), uniqueTestID)

	templateNames, err = getAllLaunchTemplates(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Templates")
	}

	assert.Contains(t, awsgo.StringValueSlice(templateNames), uniqueTestID)
}

func TestNukeLaunchTemplates(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	svc := ec2.New(session)

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	uniqueTestID := "cloud-nuke-test-" + util.UniqueID()
	createTestLaunchTemplate(t, session, uniqueTestID)

	_, err = svc.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []*string{&uniqueTestID},
	})

	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	if err := nukeAllLaunchTemplates(session, []*string{&uniqueTestID}); err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	groupNames, err := getAllLaunchTemplates(session, region, time.Now().Add(1*time.Hour), config.Config{})
	if err != nil {
		assert.Fail(t, "Unable to fetch list of Launch Templates")
	}

	assert.NotContains(t, awsgo.StringValueSlice(groupNames), uniqueTestID)
}

func TestShouldIncludeLaunchTemplate(t *testing.T) {
	mockLaunchTemplate := &ec2.LaunchTemplate{
		LaunchTemplateName: awsgo.String("cloud-nuke-test"),
		CreateTime:         awsgo.Time(time.Now()),
	}

	mockExpression, err := regexp.Compile("^cloud-nuke-*")
	if err != nil {
		logging.Logger.Fatalf("There was an error compiling regex expression %v", err)
	}

	mockExcludeConfig := config.Config{
		LaunchTemplate: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	mockIncludeConfig := config.Config{
		LaunchTemplate: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *mockExpression,
					},
				},
			},
		},
	}

	cases := []struct {
		Name           string
		LaunchTemplate *ec2.LaunchTemplate
		Config         config.Config
		ExcludeAfter   time.Time
		Expected       bool
	}{
		{
			Name:           "ConfigExclude",
			LaunchTemplate: mockLaunchTemplate,
			Config:         mockExcludeConfig,
			ExcludeAfter:   time.Now().Add(1 * time.Hour),
			Expected:       false,
		},
		{
			Name:           "ConfigInclude",
			LaunchTemplate: mockLaunchTemplate,
			Config:         mockIncludeConfig,
			ExcludeAfter:   time.Now().Add(1 * time.Hour),
			Expected:       true,
		},
		{
			Name:           "NotOlderThan",
			LaunchTemplate: mockLaunchTemplate,
			Config:         config.Config{},
			ExcludeAfter:   time.Now().Add(1 * time.Hour * -1),
			Expected:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result := shouldIncludeLaunchTemplate(c.LaunchTemplate, c.ExcludeAfter, c.Config)
			assert.Equal(t, c.Expected, result)
		})
	}
}
