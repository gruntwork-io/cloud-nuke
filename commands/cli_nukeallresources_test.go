package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/cloud-nuke/aws"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/gruntwork-io/terratest/modules/collections"
	"github.com/stretchr/testify/require"
)

// createIAMRoleArgs bundles up aws.CreateIAMRole args
type createIAMRoleArgs struct {
	awsSession               *session.Session
	roleName                 string
	roleDescription          string
	assumeRolePolicyDocument string
}

// createIAMRolePolicyArgs bundles up aws.CreateIAMRolePolicy args
type createIAMRolePolicyArgs struct {
	awsSession     *session.Session
	roleName       string
	policyName     string
	policyDocument string
}

// createTestResourcesArgs represents part of createTestResources args
type createTestResourcesArgs struct {
	resourceType string
	name         string
	tags         []map[string]string
}

func TestMain(m *testing.M) {
	err := aws.SetEnvLogLevel()
	if err != nil {
		logging.Logger.Errorf("Invalid log level - %s", err)
		os.Exit(1)
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func resourceExists(targetRegion string, targetResourceType string, targetResourceName string, account *aws.AwsAccountResources) bool {
	for region, accountResources := range account.Resources {
		if region != targetRegion {
			continue
		}
		for _, resourceCollectionObj := range accountResources.Resources {
			if resourceCollectionObj.ResourceName() == targetResourceType {
				logging.Logger.Debugf(
					"region - %s resourcetype - %s resourceIdentifiers - %+v",
					region,
					resourceCollectionObj.ResourceName(),
					resourceCollectionObj.ResourceIdentifiers(),
				)
				return collections.ListContains(resourceCollectionObj.ResourceIdentifiers(), targetResourceName)
			}
		}
	}
	return false
}

func createTestRole(awsSession *session.Session, roleArgs createIAMRoleArgs, policyArgs createIAMRolePolicyArgs) (string, error) {
	if roleArgs.roleName == "" {
		return "", nil
	}

	roleArn, err := aws.CreateIAMRole(
		awsSession,
		roleArgs.roleName,
		roleArgs.roleDescription,
		roleArgs.assumeRolePolicyDocument,
	)
	if err != nil {
		return "", err
	}

	err = aws.CreateIAMRolePolicy(
		awsSession,
		policyArgs.roleName,
		policyArgs.policyName,
		policyArgs.policyDocument,
	)
	return roleArn, err
}

func deleteTestRole(awsSession *session.Session, roleArgs createIAMRoleArgs, policyArgs createIAMRolePolicyArgs) error {
	if roleArgs.roleName == "" {
		return nil
	}
	err := aws.DeleteIAMRolePolicy(awsSession, roleArgs.roleName, policyArgs.policyName)
	if err != nil {
		return err
	}

	err = aws.DeleteIAMRole(awsSession, policyArgs.roleName)
	return err
}

func createTestResources(awsSession *session.Session, args []createTestResourcesArgs) (map[string]string, error) {
	resourceTypeIdentifierMap := map[string]string{}
	for _, arg := range args {
		if arg.resourceType == "s3" {
			svc := s3.New(awsSession)
			if err := aws.S3CreateBucket(svc, arg.name, arg.tags, false); err != nil {
				return resourceTypeIdentifierMap, err
			}
			resourceTypeIdentifierMap[arg.resourceType] = arg.name
		} else if arg.resourceType == "ec2" {
			instance, err := aws.CreateTestEC2Instance(awsSession, arg.name, false)
			if err != nil {
				return resourceTypeIdentifierMap, err
			}
			instanceID := *instance.InstanceId
			resourceTypeIdentifierMap[arg.resourceType] = instanceID
		}
		logging.Logger.Debugf("Created test %s - %s", arg.resourceType, arg.name)
	}
	return resourceTypeIdentifierMap, nil
}

func buildNukeAllResourcesArgs(awsSession *session.Session, resourceTypeIdentifierMap map[string]string, targetRegion string, ignoreErrResourceTypes []string) aws.NukeAllResourcesArgs {
	nukeAccount := aws.AwsAccountResources{
		Resources: make(map[string]aws.AwsRegionResource),
	}
	resourcesInRegion := aws.AwsRegionResource{}

	resourceIdentifier, ok := resourceTypeIdentifierMap["s3"]
	if ok {
		s3Buckets := aws.S3Buckets{}
		s3Buckets.Names = []string{resourceIdentifier}
		resourcesInRegion.Resources = append(resourcesInRegion.Resources, s3Buckets)
	}

	resourceIdentifier, ok = resourceTypeIdentifierMap["ec2"]
	if ok {
		ec2Instances := aws.EC2Instances{}
		ec2Instances.InstanceIds = []string{resourceIdentifier}
		resourcesInRegion.Resources = append(resourcesInRegion.Resources, ec2Instances)
	}

	if len(resourcesInRegion.Resources) > 0 {
		nukeAccount.Resources[targetRegion] = resourcesInRegion
		return aws.NukeAllResourcesArgs{
			Account:                &nukeAccount,
			Regions:                []string{targetRegion},
			IgnoreErrResourceTypes: ignoreErrResourceTypes,
			RegionSessionMap:       map[string]*session.Session{targetRegion: awsSession},
		}
	}
	return aws.NukeAllResourcesArgs{}
}

// TestNukeAllResourcesAllPerms tests deletion of test S3 and EC2 instance with user's
// default AWS session
func testNukeAllResources(t *testing.T, awsParams aws.CloudNukeAWSParams, assumeRoleArn string, args testNukeAllResourcesArgs) {
	// Generate test bucket and test instance name
	bucketName := aws.S3GenBucketName()
	ec2InstanceName := "cloud-nuke-test-" + util.UniqueID()

	resourceTypeIdentifierMap := map[string]string{}
	resourceTypes := []string{}
	tags := []map[string]string{{"Key": "cloud-nuke-test", "Value": "true"}}

	// Create test bucket and test instance
	resourceTypeIdentifierMap, err := createTestResources(
		awsParams.AWSSession,
		[]createTestResourcesArgs{
			{resourceType: "s3", name: bucketName, tags: tags},
			{resourceType: "ec2", name: ec2InstanceName, tags: tags},
		},
	)

	// Ensure test bucket + test EC2 instance gets deleted at the end of the test
	// Add defer before checking error to handle the case where partial resources get
	// created and must be destroyed
	defer aws.NukeAllResources(buildNukeAllResourcesArgs(awsParams.AWSSession, resourceTypeIdentifierMap, awsParams.Region, args.ignoreErrResourceTypes))

	require.NoError(t, err, "Failed to create test resources")

	for resourceType := range resourceTypeIdentifierMap {
		resourceTypes = append(resourceTypes, resourceType)
	}

	// Assume role
	var roleSession *session.Session
	if assumeRoleArn == "" {
		roleSession = awsParams.AWSSession
	} else {
		roleSession, err = aws.AssumeIAMRole(assumeRoleArn, awsParams.Region)
		require.NoErrorf(t, err, "Failed to assume role - %s - %s", assumeRoleArn, err)
	}

	excludeAfter, err := parseDurationParam("0s")
	require.NoError(t, err, "parseDurationParam failed")

	// Do assumed role get for existing resources of type EC2 and S3
	account, err := aws.GetAllResources(
		aws.GetAllResourcesArgs{
			TargetRegions:          []string{awsParams.Region},
			ExcludeAfter:           *excludeAfter,
			NukeResourceTypes:      resourceTypes,
			IgnoreErrResourceTypes: args.ignoreErrResourceTypes,
			RegionSessionMap:       map[string]*session.Session{awsParams.Region: roleSession},
		},
	)
	// Validate that get call should fail if specified
	if args.getResourcesShouldFail {
		require.Error(t, err, "GetAllResources did not fail")
		logging.Logger.Debug("Expected: GetAllResources failed")
		return
	}
	require.NoError(t, err, "GetAllResources failed")
	logging.Logger.Debug("Expected: GetAllResources passed")

	// Validate that get call should return the right resource types
	if collections.ListContains(args.getResourcesOutputTypes, "ec2") {
		require.Truef(
			t, resourceExists(awsParams.Region, "ec2", resourceTypeIdentifierMap["ec2"], account),
			"Failed to find test EC2 instance - name - %s - instanceID - %s", ec2InstanceName, resourceTypeIdentifierMap["ec2"],
		)
		logging.Logger.Debug("Expected: Post get check: ec2 exists")
	}
	if collections.ListContains(args.getResourcesOutputTypes, "s3") {
		require.Truef(
			t, resourceExists(awsParams.Region, "s3", bucketName, account),
			"Failed to find test S3 bucket - %s", bucketName,
		)
		logging.Logger.Debug("Expected: Post get check: s3 exists")
	}

	// Do assumed role nuke for target test resources
	err = aws.NukeAllResources(buildNukeAllResourcesArgs(roleSession, resourceTypeIdentifierMap, awsParams.Region, args.ignoreErrResourceTypes))

	// Validate that nuke call should fail if specified
	if args.nukeResourcesShouldFail {
		require.Error(t, err, "assumedRole NukeAllResources did not fail")
		logging.Logger.Debug("Expected: NukeAllResources failed")
		return
	}
	require.NoError(t, err, "assumedRole NukeAllResources failed")
	logging.Logger.Debug("Expected: NukeAllResources passed")

	// Post deletion - get existing resources of type EC2 and S3
	account, err = aws.GetAllResources(
		aws.GetAllResourcesArgs{
			TargetRegions:          []string{awsParams.Region},
			ExcludeAfter:           *excludeAfter,
			NukeResourceTypes:      resourceTypes,
			IgnoreErrResourceTypes: args.ignoreErrResourceTypes,
			RegionSessionMap:       map[string]*session.Session{awsParams.Region: awsParams.AWSSession},
		},
	)
	require.NoError(t, err, "Failed to get test resources")
	logging.Logger.Debug("Expected: GetAllResources passed")

	// Validate post nuke resources
	ec2Exists := resourceExists(awsParams.Region, "ec2", resourceTypeIdentifierMap["ec2"], account)
	s3Exists := resourceExists(awsParams.Region, "s3", resourceTypeIdentifierMap["s3"], account)

	if collections.ListContains(args.postNukeResources, "ec2") {
		require.Truef(
			t, ec2Exists,
			"Test EC2 instance deleted when it should not have - name - %s - instanceID - %s",
			ec2InstanceName, resourceTypeIdentifierMap["ec2"],
		)
		logging.Logger.Debug("Expected: Post nuke check - ec2 exists")
	} else {
		require.Falsef(
			t, ec2Exists,
			"Test EC2 instance not deleted - name - %s - instanceId - %s",
			ec2InstanceName, resourceTypeIdentifierMap["ec2"],
		)
		logging.Logger.Debug("Expected: Post nuke check - ec2 deleted")
	}
	if collections.ListContains(args.postNukeResources, "s3") {
		require.Truef(t, s3Exists, "Test S3 bucket deleted when it should not have - %s", bucketName)
		logging.Logger.Debug("Expected: Post nuke check - s3 exists")
	} else {
		require.Falsef(t, s3Exists, "Test S3 bucket not deleted - %s", bucketName)
		logging.Logger.Debug("Expected: Post nuke check - s3 deleted")
	}
}

// testNukeAllResourcesArgs bundles up testNukeAllResources args
type testNukeAllResourcesArgs struct {
	awsSession              *session.Session // main AWS session which will be used to create/delete test resources
	assumeRoleArn           string           // ARN of role to assume
	ignoreErrResourceTypes  []string         // list of resource types to ignore AWS errors
	getResourcesShouldFail  bool             // whether GetAllResources should return err
	getResourcesOutputTypes []string         // what resourcetypes from test resources should GetAllResources return
	nukeResourcesShouldFail bool             // flag which decides whether NukeAllResources should return err
	postNukeResources       []string         // what resourcetypes from test resources should exist after NukeAllResources
}

// getIAMRolePolicyMamp maps a role type to a corresponding policy
func getTestRolePolicyMap() map[string]string {
	return map[string]string{
		"NoPerms": aws.TrimPolicyDocument(`{
			"Version": "2012-10-17",
			"Statement": {
				"Effect": "Allow",
				"Action": "ec2:DescribeRegions",
				"Resource": "*"
			}
		}`),
		"ROPerms": aws.TrimPolicyDocument(`{
			"Version": "2012-10-17",
			"Statement": {
				"Effect": "Allow",
				"Action": [
					"ec2:Describe*",
					"s3:List*",
					"s3:Get*"
				],
				"Resource": "*"
			}
		}`),
		"EC2ROS3RWPerms": aws.TrimPolicyDocument(`{
			"Version": "2012-10-17",
			"Statement": {
				"Effect": "Allow",
				"Action": [
					"ec2:Describe*",
					"s3:*"
				],
				"Resource": "*"
			}
		}`),
	}
}

// TestNukeAllResourcesNoPerms tests deletion of test S3 and EC2 instance with an assumed
// role which does not have S3/EC2 read/write permissions
func TestNukeAllResources(t *testing.T) {
	t.Parallel()
	// Create a top level AWS session object which will be used to create/destroy roles
	// and resources. Specifically use us-east-1 as we are running >= 8 tests and vCpu limits
	// for eu-west-3 and some other regions are around - 8 - which allow only 4 instances to be created
	awsParams, err := aws.NewCloudNukeAWSParams("us-east-1")
	require.NoError(t, err, "Failed to setup AWS params")
	testRolePolicyMap := getTestRolePolicyMap()

	// Static assume role policy document which will allow the current identity to assume
	// test roles
	callerIdentityArn, err := aws.GetCallerIdentityArn(awsParams.AWSSession)
	require.NoError(t, err, "Failed to get AWS caller identity")
	assumeRolePolicyDocumentTmpl := aws.TrimPolicyDocument(`{
		"Version": "2012-10-17",
			"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": "$callerIdentityArn"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`)
	assumeRolePolicyDocument := strings.Replace(
		assumeRolePolicyDocumentTmpl, "$callerIdentityArn", callerIdentityArn, -1,
	)

	var testCases = []struct {
		name                    string
		createIAMRoleArgs       createIAMRoleArgs
		createIAMRolePolicyArgs createIAMRolePolicyArgs
		args                    testNukeAllResourcesArgs
	}{
		{
			"AllPerms",
			createIAMRoleArgs{
				roleName: "", // Do not create any role
			},
			createIAMRolePolicyArgs{},
			testNukeAllResourcesArgs{
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{"ec2", "s3"},
				nukeResourcesShouldFail: false,
				postNukeResources:       []string{},
			},
		},
		{
			"NoPerms",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-1-noperms",
				roleDescription:          "cloud-nuke-test role for NoPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-1-noperms",
				policyName:     "cloud-nuke-test-1-noperms-policy",
				policyDocument: testRolePolicyMap["NoPerms"],
			},
			testNukeAllResourcesArgs{
				getResourcesShouldFail:  true,
				getResourcesOutputTypes: []string{},
			},
		},
		{
			"NoPermsIgnoreErrors",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-2-noperms",
				roleDescription:          "cloud-nuke-test role for NoPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-2-noperms",
				policyName:     "cloud-nuke-test-2-noperms-policy",
				policyDocument: testRolePolicyMap["NoPerms"],
			},
			testNukeAllResourcesArgs{
				ignoreErrResourceTypes:  []string{"ec2", "s3"},
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{},
				nukeResourcesShouldFail: false,
				postNukeResources:       []string{"ec2", "s3"},
			},
		},
		{
			"NoPermsIgnoreErrorsOnlyForS3",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-3-noperms",
				roleDescription:          "cloud-nuke-test role for noperms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-3-noperms",
				policyName:     "cloud-nuke-test-3-noperms-policy",
				policyDocument: testRolePolicyMap["NoPerms"],
			},
			testNukeAllResourcesArgs{
				ignoreErrResourceTypes: []string{"s3"},
				// GetAllResourcse - EC2 is checked before S3 - without ignoreErrors for EC2
				// - the function will return failure before checking and ignoring S3
				getResourcesShouldFail:  true,
				getResourcesOutputTypes: []string{},
			},
		},
		{
			"ROPerms",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-4-roperms",
				roleDescription:          "cloud-nuke-test role for ROPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-4-roperms",
				policyName:     "cloud-nuke-test-4-roperms-policy",
				policyDocument: testRolePolicyMap["ROPerms"],
			},
			testNukeAllResourcesArgs{
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{"ec2", "s3"},
				nukeResourcesShouldFail: true,
				postNukeResources:       []string{"ec2", "s3"},
			},
		},
		{
			"ROPermsIgnoreErrors",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-5-roperms",
				roleDescription:          "cloud-nuke-test role for ROPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-5-roperms",
				policyName:     "cloud-nuke-test-5-roperms-policy",
				policyDocument: testRolePolicyMap["ROPerms"],
			},
			testNukeAllResourcesArgs{
				ignoreErrResourceTypes:  []string{"ec2", "s3"},
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{"ec2", "s3"},
				nukeResourcesShouldFail: false,
				postNukeResources:       []string{"ec2", "s3"},
			},
		},
		{
			"EC2ROS3RWPerms",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-6-ec2ros3rwperms",
				roleDescription:          "cloud-nuke-test role for EC2ROS3RWPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-6-ec2ros3rwperms",
				policyName:     "cloud-nuke-test-6-ec2ros3rwperms-policy",
				policyDocument: testRolePolicyMap["EC2ROS3RWPerms"],
			},
			testNukeAllResourcesArgs{
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{"ec2", "s3"},
				nukeResourcesShouldFail: true,
				postNukeResources:       []string{"ec2"},
			},
		},
		{
			"EC2ROS3RWPermsIgnoreErrors",
			createIAMRoleArgs{
				roleName:                 "cloud-nuke-test-7-ec2ros3rwperms",
				roleDescription:          "cloud-nuke-test role for EC2ROS3RWPerms test",
				assumeRolePolicyDocument: assumeRolePolicyDocument,
			},
			createIAMRolePolicyArgs{
				roleName:       "cloud-nuke-test-7-ec2ros3rwperms",
				policyName:     "cloud-nuke-test-7-ec2ros3rwperms-policy",
				policyDocument: testRolePolicyMap["EC2ROS3RWPerms"],
			},
			testNukeAllResourcesArgs{
				ignoreErrResourceTypes:  []string{"ec2", "s3"},
				getResourcesShouldFail:  false,
				getResourcesOutputTypes: []string{"ec2", "s3"},
				nukeResourcesShouldFail: false,
				postNukeResources:       []string{"ec2"},
			},
		},
	}
	for _, tc := range testCases {
		// Capture the range variable as per https://blog.golang.org/subtests
		// Not doing this will lead to tc being set to the last entry in the testCases
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			roleArn, err := createTestRole(awsParams.AWSSession, tc.createIAMRoleArgs, tc.createIAMRolePolicyArgs)
			require.NoErrorf(t, err, "Failed to setup test role - %s", err)
			if roleArn != "" {
				logging.Logger.Debugf("Created test role - %s", roleArn)
			}

			testNukeAllResources(t, awsParams, roleArn, tc.args)

			err = deleteTestRole(awsParams.AWSSession, tc.createIAMRoleArgs, tc.createIAMRolePolicyArgs)
			require.NoErrorf(t, err, "Failed to teardown test role - %s", err)
			if roleArn != "" {
				logging.Logger.Debugf("Deleted test role - %s", roleArn)
			}
		})
	}
}
