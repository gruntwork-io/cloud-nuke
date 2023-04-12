package aws

import (
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/telemetry"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Stores info for cleanup
type entityInfo struct {
	PolicyArn *string
	UserName  *string
	GroupName *string
	RoleName  *string
}

const fakePolicy string = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "VisualEditor0",
				"Effect": "Allow",
				"Action": "elasticmapreduce:*",
				"Resource": "*",
				"Condition": {
					"BoolIfExists": {
						"aws:MultiFactorAuthPresent": "true"
					}
				}
			}
		]
	}`

func TestListIamPolicies(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()
	region, err := getRandomRegion()
	require.NoError(t, err)

	localSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	policyArns, err := getAllLocalIamPolicies(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotEmpty(t, policyArns)
}

func createPolicyWithNoEntities(t *testing.T, session *session.Session, name string) (string, error) {
	svc := iam.New(session)
	doc := awsgo.String(fakePolicy)
	policy, err := svc.CreatePolicy(&iam.CreatePolicyInput{PolicyName: awsgo.String(name), PolicyDocument: doc})
	require.NoError(t, err)
	return *policy.Policy.Arn, nil
}

func createPolicyWithEntities(t *testing.T, session *session.Session, name string) (*entityInfo, error) {
	policyArn, err := createPolicyWithNoEntities(t, session, name)
	svc := iam.New(session)
	if err != nil {
		return nil, err
	}

	//Create version
	versionInput := &iam.CreatePolicyVersionInput{
		PolicyArn:      awsgo.String(policyArn),
		PolicyDocument: awsgo.String(fakePolicy),
	}
	_, err = svc.CreatePolicyVersion(versionInput)
	if err != nil {
		return nil, err
	}
	//Create User and link
	userName := awsgo.String("test-user-" + util.UniqueID())
	_, err = svc.CreateUser(&iam.CreateUserInput{UserName: userName})
	if err != nil {
		return nil, err
	}

	userPolicyLink := &iam.AttachUserPolicyInput{
		UserName:  userName,
		PolicyArn: awsgo.String(policyArn),
	}
	_, err = svc.AttachUserPolicy(userPolicyLink)
	if err != nil {
		return nil, err
	}

	//Create role and link
	roleName := awsgo.String("test-role-" + util.UniqueID())
	roleInput := &iam.CreateRoleInput{
		RoleName: roleName,
		AssumeRolePolicyDocument: awsgo.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"sts:AssumeRole"
					],
					"Principal": {
						"Service": [
							"ec2.amazonaws.com"
						]
					}
				}
			]
		}`),
	}
	_, err = svc.CreateRole(roleInput)
	if err != nil {
		return nil, err
	}

	rolePolicyLink := &iam.AttachRolePolicyInput{RoleName: roleName, PolicyArn: awsgo.String(policyArn)}
	_, err = svc.AttachRolePolicy(rolePolicyLink)
	if err != nil {
		return nil, err
	}

	//Create group and link
	groupName := awsgo.String("test-group-" + util.UniqueID())
	groupInput := &iam.CreateGroupInput{GroupName: groupName}
	_, err = svc.CreateGroup(groupInput)
	if err != nil {
		return nil, err
	}

	groupPolicyLink := &iam.AttachGroupPolicyInput{GroupName: groupName, PolicyArn: awsgo.String(policyArn)}
	_, err = svc.AttachGroupPolicy(groupPolicyLink)
	if err != nil {
		return nil, err
	}

	return &entityInfo{
		PolicyArn: &policyArn,
		UserName:  userName,
		RoleName:  roleName,
		GroupName: groupName,
	}, nil
}

func deletePolicyExtraResources(session *session.Session, entityInfo *entityInfo) {
	svc := iam.New(session)
	//Delete User
	_, err := svc.DeleteUser(&iam.DeleteUserInput{UserName: entityInfo.UserName})
	if err != nil {
		logging.Logger.Errorf("Unable to delete test user: %s", *entityInfo.UserName)
	}

	//Delete Role
	_, err = svc.DeleteRole(&iam.DeleteRoleInput{RoleName: entityInfo.RoleName})
	if err != nil {
		logging.Logger.Errorf("Unable to delete test role: %s", *entityInfo.RoleName)
	}

	//Delete Group
	_, err = svc.DeleteGroup(&iam.DeleteGroupInput{GroupName: entityInfo.GroupName})
	if err != nil {
		logging.Logger.Errorf("Unable to delete test group: %s", *entityInfo.GroupName)
	}
}

func TestNukeIamPolicies(t *testing.T) {
	telemetry.InitTelemetry("cloud-nuke", "", "")
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	localSession, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region),
	})
	require.NoError(t, err)

	//Create test entities
	emptyName := "cloud-nuke-test" + util.UniqueID()
	var emptyPolicyArn string
	emptyPolicyArn, err = createPolicyWithNoEntities(t, localSession, emptyName)
	require.NoError(t, err)

	nonEmptyName := "cloud-nuke-test" + util.UniqueID()
	entities, err := createPolicyWithEntities(t, localSession, nonEmptyName)
	defer deletePolicyExtraResources(localSession, entities)
	require.NoError(t, err)

	//Assert test entities exist
	var policyArns []*string
	policyArns, err = getAllLocalIamPolicies(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(policyArns), emptyPolicyArn)
	assert.Contains(t, awsgo.StringValueSlice(policyArns), *entities.PolicyArn)

	//Nuke test entities
	err = nukeAllIamPolicies(localSession, []*string{&emptyPolicyArn, entities.PolicyArn})
	require.NoError(t, err)

	//Assert test entities don't exist anymore
	policyArns, err = getAllLocalIamPolicies(localSession, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(policyArns), emptyPolicyArn)
	assert.NotContains(t, awsgo.StringValueSlice(policyArns), *entities.PolicyArn)
}
