package aws

import (
	"fmt"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codedeploy"
	"github.com/gruntwork-io/cloud-nuke/config"
)

func randomString() string {
	return time.Now().Format("20060102150405")
}

func createSession(region string) (*session.Session, error) {
	return session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
}

func createCodeDeployApplication(session *session.Session, applicationName, region string) error {
	svc := codedeploy.New(session)
	computePlatform := selectComputePlatform()
	_, err := svc.CreateApplication(
		&codedeploy.CreateApplicationInput{
			ApplicationName: &applicationName,
			ComputePlatform: &computePlatform,
		})

	return err
}

func selectComputePlatform() string {
	computePlatforms := []string{"Server", "Lambda", "ECS"}
	return computePlatforms[rand.Intn(len(computePlatforms))]
}

func createCodeDeployTestEnvironment(numberOfApplications int, namePostfix string) (*session.Session, []string, error) {
	region, err := getRandomRegion()
	if err != nil {
		return nil, nil, err
	}

	session, err := createSession(region)
	if err != nil {
		return nil, nil, err
	}

	identifiers := make([]string, 0, numberOfApplications)

	for i := 0; i < numberOfApplications; i++ {
		applicationName := fmt.Sprintf("cloud-nuke-test-%s-%d", namePostfix, i)
		err := createCodeDeployApplication(session, applicationName, *session.Config.Region)
		if err != nil {
			return nil, nil, err
		}
		identifiers = append(identifiers, applicationName)
	}

	return session, identifiers, nil
}

func TestGetAllCodeDeployApplicationsSimple(t *testing.T) {
	namePostfix := randomString()
	session, identifiers, err := createCodeDeployTestEnvironment(5, namePostfix)
	if err != nil {
		t.Fatalf("Failed to create CodeDeploy test environment: %v", err)
	}
	defer nukeAllCodeDeployApplications(session, identifiers)

	// Test that we can get all CodeDeploy Applications
	applicationNames, err := getAllCodeDeployApplications(session, time.Now(), config.Config{})
	if err != nil {
		t.Fatalf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != 5 {
		t.Fatalf("Expected 5 CodeDeploy Applications, got %d: %v", len(applicationNames), applicationNames)
	}
}

func TestGetAllCodeDeployApplicationsFilteredCreationDate(t *testing.T) {
	namePostfix := randomString()
	session, identifiers, err := createCodeDeployTestEnvironment(5, namePostfix)
	if err != nil {
		t.Errorf("Failed to create CodeDeploy test environment: %v", err)
	}
	defer nukeAllCodeDeployApplications(session, identifiers)

	// Test that we can get all CodeDeploy Applications
	applicationNames, err := getAllCodeDeployApplications(session, time.Now().AddDate(0, 0, -1), config.Config{})
	if err != nil {
		t.Errorf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != 0 {
		t.Errorf("Expected 0 CodeDeploy Applications, got %d: %v", len(applicationNames), applicationNames)
	}
}

func TestGetAllCodeDeployApplicationsIncludedByName(t *testing.T) {
	namePostfix := randomString()
	session, identifiers, err := createCodeDeployTestEnvironment(5, namePostfix)
	if err != nil {
		t.Errorf("Failed to create CodeDeploy test environment: %v", err)
	}
	defer nukeAllCodeDeployApplications(session, identifiers)

	// Test that we can get all CodeDeploy Applications
	applicationNames, err := getAllCodeDeployApplications(session, time.Now(), config.Config{
		CodeDeployApplications: config.ResourceType{
			IncludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *regexp.MustCompile(fmt.Sprintf("cloud-nuke-test-%s-1", namePostfix)),
					},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != 1 {
		t.Errorf("Expected 1 CodeDeploy Application, got %d: %v", len(applicationNames), applicationNames)
	}
}

func TestGetAllCodeDeployApplicationsExcludedByName(t *testing.T) {
	namePostfix := randomString()
	session, identifiers, err := createCodeDeployTestEnvironment(5, namePostfix)
	if err != nil {
		t.Errorf("Failed to create CodeDeploy test environment: %v", err)
	}
	defer nukeAllCodeDeployApplications(session, identifiers)

	// Test that we can get all CodeDeploy Applications
	applicationNames, err := getAllCodeDeployApplications(session, time.Now(), config.Config{
		CodeDeployApplications: config.ResourceType{
			ExcludeRule: config.FilterRule{
				NamesRegExp: []config.Expression{
					{
						RE: *regexp.MustCompile(fmt.Sprintf("cloud-nuke-test-%s-1", namePostfix)),
					},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != 4 {
		t.Errorf("Expected 4 CodeDeploy Application, got %d: %v", len(applicationNames), applicationNames)
	}
}

func TestNukeAllCodeDeployApplications(t *testing.T) {
	namePostfix := randomString()

	// note we set 105 applications here to ensure pagination and batching is working.
	applicationCount := 105

	session, identifiers, err := createCodeDeployTestEnvironment(applicationCount, namePostfix)
	if err != nil {
		t.Errorf("Failed to create CodeDeploy test environment: %v", err)
	}

	// ensure we leave the test environment clean
	defer nukeAllCodeDeployApplications(session, identifiers)

	// Test that all CodeDeploy Applications are found
	applicationNames, err := getAllCodeDeployApplications(session, time.Now(), config.Config{})
	if err != nil {
		t.Errorf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != applicationCount {
		t.Errorf("Expected %d CodeDeploy Applications, got %d: %v", applicationCount, len(applicationNames), applicationNames)
	}

	// Nuke all CodeDeploy Applications
	err = nukeAllCodeDeployApplications(session, applicationNames)
	if err != nil {
		t.Errorf("Failed to nuke CodeDeploy Applications: %v", err)
	}

	// Test that all CodeDeploy Applications are gone
	applicationNames, err = getAllCodeDeployApplications(session, time.Now(), config.Config{})
	if err != nil {
		t.Errorf("Failed to get CodeDeploy Applications: %v", err)
	}

	if len(applicationNames) != 0 {
		t.Errorf("Expected 0 CodeDeploy Applications, got %d: %v", len(applicationNames), applicationNames)
	}
}
