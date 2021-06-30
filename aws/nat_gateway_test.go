package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	terraws "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListNatGateways(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ec2.New(session)

	ngwID := createNatGateway(t, svc, region)
	defer deleteNatGateway(t, svc, ngwID, true)

	natGatewayIDs, err := getAllNatGateways(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(natGatewayIDs), aws.StringValue(ngwID))
}

func TestTimeFilterExclusionNewlyCreatedNatGateway(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ec2.New(session)

	// Creates a NGW
	ngwID := createNatGateway(t, svc, region)
	defer deleteNatGateway(t, svc, ngwID, true)

	// Assert NGW is picked up without filters
	natGatewayIDsNewer, err := getAllNatGateways(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, aws.StringValueSlice(natGatewayIDsNewer), aws.StringValue(ngwID))

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	natGatewayIDsOlder, err := getAllNatGateways(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, aws.StringValueSlice(natGatewayIDsOlder), aws.StringValue(ngwID))
}

func TestNukeNatGatewayOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ec2.New(session)

	// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
	ngwID := createNatGateway(t, svc, region)
	defer deleteNatGateway(t, svc, ngwID, false)
	identifiers := []*string{ngwID}

	require.NoError(
		t,
		nukeAllNatGateways(session, identifiers),
	)

	// Make sure the NAT gateway is deleted.
	assertNatGatewaysDeleted(t, svc, identifiers)
}

func TestNukeNatGatewayMoreThanOne(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	require.NoError(t, err)
	svc := ec2.New(session)

	natGateways := []*string{}
	for i := 0; i < 3; i++ {
		// We ignore errors in the delete call here, because it is intended to be a stop gap in case there is a bug in nuke.
		ngwID := createNatGateway(t, svc, region)
		defer deleteNatGateway(t, svc, ngwID, false)
		natGateways = append(natGateways, ngwID)
	}

	require.NoError(
		t,
		nukeAllNatGateways(session, natGateways),
	)

	// Make sure the NAT Gateway is deleted.
	assertNatGatewaysDeleted(t, svc, natGateways)
}

// Helper functions for driving the NAT gateway tests

// createNatGateway will create a new NAT gateway in the default VPC
func createNatGateway(t *testing.T, svc *ec2.EC2, region string) *string {
	defaultVpc := terraws.GetDefaultVpc(t, region)
	subnet := defaultVpc.Subnets[0]

	resp, err := svc.CreateNatGateway(&ec2.CreateNatGatewayInput{
		SubnetId:         aws.String(subnet.Id),
		ConnectivityType: aws.String(ec2.ConnectivityTypePrivate),
	})
	require.NoError(t, err)
	if resp.NatGateway == nil {
		t.Fatalf("Impossible error: AWS returned nil NAT gateway")
	}
	return resp.NatGateway.NatGatewayId
}

// deleteNatGateway is a function to delete the given NAT gateway.
func deleteNatGateway(t *testing.T, svc *ec2.EC2, ngwID *string, checkErr bool) {
	input := &ec2.DeleteNatGatewayInput{NatGatewayId: ngwID}
	_, err := svc.DeleteNatGateway(input)
	if checkErr {
		require.NoError(t, err)
	}
}

func assertNatGatewaysDeleted(t *testing.T, svc *ec2.EC2, identifiers []*string) {
	resp, err := svc.DescribeNatGateways(&ec2.DescribeNatGatewaysInput{NatGatewayIds: identifiers})
	require.NoError(t, err)
	if len(resp.NatGateways) == 0 {
		return
	}
	if len(resp.NatGateways) > len(identifiers) {
		t.Fatalf("More than expected %d NAT gateway (found %d) for query", len(identifiers), len(resp.NatGateways))
	}
	for _, ngw := range resp.NatGateways {
		if ngw == nil {
			continue
		}
		if aws.StringValue(ngw.State) != ec2.NatGatewayStateDeleted {
			t.Fatalf("NAT Gateway not deleted by nuke operation")
		}
	}
}
