package aws

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsgo "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/logging"
	"github.com/gruntwork-io/cloud-nuke/util"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestListIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)

	assert.NotEmpty(t, userNames)
}

func createTestUser(t *testing.T, session *session.Session, name string) error {
	svc := iam.New(session)
	input := &iam.CreateUserInput{
		UserName: aws.String(name),
	}

	_, err := svc.CreateUser(input)
	require.NoError(t, err)

	return nil
}

func TestCreateIamUser(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)

	err = createTestUser(t, session, name)
	defer nukeAllIamUsers(session, []*string{&name})
	require.NoError(t, err)

	userNames, err = getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(userNames), name)
}

func TestNukeIamUsers(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	name := "cloud-nuke-test-" + util.UniqueID()
	err = createTestUser(t, session, name)
	require.NoError(t, err)

	err = nukeAllIamUsers(session, []*string{&name})
	require.NoError(t, err)
}

func TestTimeFilterExclusionNewlyCreatedIamUser(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	// Assert user didn't exist
	name := "cloud-nuke-test-" + util.UniqueID()
	userNames, err := getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)

	// Creates a user
	err = createTestUser(t, session, name)
	defer nukeAllIamUsers(session, []*string{&name})

	// Assert user is created
	userNames, err = getAllIamUsers(session, time.Now(), config.Config{})
	require.NoError(t, err)
	assert.Contains(t, awsgo.StringValueSlice(userNames), name)

	// Assert user doesn't appear when we look at users older than 1 Hour
	olderThan := time.Now().Add(-1 * time.Hour)
	userNames, err = getAllIamUsers(session, olderThan, config.Config{})
	require.NoError(t, err)
	assert.NotContains(t, awsgo.StringValueSlice(userNames), name)
}

// We need to create a valid X.509 certificate and upload it to associate it with a "Signing Certificate" for our user
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/set-up-ami-tools.html?icmpid=docs_iam_console#ami-tools-managing-certs
// https://medium.com/@shaneutt/create-sign-x509-certificates-in-golang-8ac4ae49f903
func createX509Certificate() (string, error) {
	// Create the Certificate Authority

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(mathRand.Int63()),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94129"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Certificate Authority's Private Key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}

	// PEM enconding the Certificate Authority's Private Key
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	// Certificate Authority's Certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return "", err
	}

	// PEM enconding the Certificate Authority's Certificate
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// ---

	// Create the actual certificate

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(mathRand.Int63()),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94129"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, 0, 1),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Certificate's Private Key
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}

	// Create the Certificate, signed by the Certificate Authority
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return "", err
	}

	// PEM encode the Certificate
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM.String(), nil
}

func createSSHPublicKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	return string(publicKeyBytes), nil
}

// Information about extra resources attached to a user
// This is used for cleaning up after the user is deleted
type userInfos struct {
	UserName  *string
	PolicyArn *string
	GroupName *string
}

// To validate that we can delete a user we need to create a user that have all the possible attached items
func createFullTestUser(t *testing.T, session *session.Session) (*userInfos, error) {
	logging.Logger.Infof("Creating temporary user to validate the IAM Users deletion")

	userName := "cloud-nuke-test-full-user-" + util.UniqueID()
	svc := iam.New(session)
	var err error

	// Create the user
	_, err = svc.CreateUser(&iam.CreateUserInput{
		UserName: aws.String(userName),
	})
	require.NoError(t, err)

	// Create Login Profile
	password := fmt.Sprintf("a1@%s%s%s", util.UniqueID(), util.UniqueID(), util.UniqueID())
	_, err = svc.CreateLoginProfile(&iam.CreateLoginProfileInput{
		Password:              aws.String(password),
		PasswordResetRequired: aws.Bool(true),
		UserName:              aws.String(userName),
	})
	require.NoError(t, err)

	// Create a User Policy
	policyOutput, err := svc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
					{
							"Sid": "VisualEditor0",
							"Effect": "Allow",
							"Action": "ec2:DescribeInstances",
							"Resource": "*"
					}
			]
		}`),
		PolicyName:  aws.String("policy-" + userName),
		Description: aws.String("Policy created by cloud-nuke tests - Should be deleted"),
	})
	require.NoError(t, err)

	// Attaching the Policy to the User
	_, err = svc.AttachUserPolicy(&iam.AttachUserPolicyInput{
		PolicyArn: policyOutput.Policy.Arn,
		UserName:  aws.String(userName),
	})
	require.NoError(t, err)

	// Create an inline user policy
	_, err = svc.PutUserPolicy(&iam.PutUserPolicyInput{
		PolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
					{
							"Sid": "VisualEditor0",
							"Effect": "Allow",
							"Action": "ec2:DescribeInstances",
							"Resource": "*"
					}
			]
		}`),
		PolicyName: aws.String("inline-policy-" + userName),
		UserName:   aws.String(userName),
	})

	// Create a Group
	groupOutput, err := svc.CreateGroup(&iam.CreateGroupInput{
		GroupName: aws.String("group-" + userName),
	})
	require.NoError(t, err)

	// Attach the user to the Group
	_, err = svc.AddUserToGroup(&iam.AddUserToGroupInput{
		GroupName: groupOutput.Group.GroupName,
		UserName:  aws.String(userName),
	})
	require.NoError(t, err)

	// Create an Access Key
	_, err = svc.CreateAccessKey(&iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	})
	require.NoError(t, err)

	// Upload a Signing Certificate
	// There's no API for creating a certificate, only for uploading one (`UploadSigningCertificate`).
	// So, we have to create a valid X.509 certificate then upload it.
	certificate, err := createX509Certificate()
	require.NoError(t, err)
	_, err = svc.UploadSigningCertificate(&iam.UploadSigningCertificateInput{
		CertificateBody: &certificate,
		UserName:        aws.String(userName),
	})
	require.NoError(t, err)

	// Upload a SSH Key
	// There's no API for creating a ssh key, only for uploading one (`UploadSSHPublicKey`).
	// So, we have to create a valid ssh key then upload it.
	sshKey, err := createSSHPublicKey()
	require.NoError(t, err)
	_, err = svc.UploadSSHPublicKey(&iam.UploadSSHPublicKeyInput{
		SSHPublicKeyBody: &sshKey,
		UserName:         aws.String(userName),
	})
	require.NoError(t, err)

	// Generate a "Service Specific Credential". CodeCommit in this case
	_, err = svc.CreateServiceSpecificCredential(&iam.CreateServiceSpecificCredentialInput{
		ServiceName: aws.String("codecommit.amazonaws.com"),
		UserName:    aws.String(userName),
	})
	require.NoError(t, err)

	// Generate a "Service Specific Credential". Amazon Keyspaces (for Cassandra) in this case
	_, err = svc.CreateServiceSpecificCredential(&iam.CreateServiceSpecificCredentialInput{
		ServiceName: aws.String("cassandra.amazonaws.com"),
		UserName:    aws.String(userName),
	})
	require.NoError(t, err)

	// Create a Virtual MFA Device for the user
	output, err := svc.CreateVirtualMFADevice(&iam.CreateVirtualMFADeviceInput{
		VirtualMFADeviceName: aws.String("mfa-" + userName),
	})
	require.NoError(t, err)
	virtualMFADevice := output.VirtualMFADevice
	// We now need to generate two codes to enable the MFA device
	authenticationCode1, err := totp.GenerateCode(string(virtualMFADevice.Base32StringSeed), time.Now())
	require.NoError(t, err)
	logging.Logger.Infof("Will sleep for 30 seconds so we can generate the second OTP for the user that is being created")
	time.Sleep(31 * time.Second) // 31 seconds just to be sure
	authenticationCode2, err := totp.GenerateCode(string(virtualMFADevice.Base32StringSeed), time.Now())
	require.NoError(t, err)
	// Associate the Virtual MFA Device to the user
	_, err = svc.EnableMFADevice(&iam.EnableMFADeviceInput{
		AuthenticationCode1: aws.String(authenticationCode1),
		AuthenticationCode2: aws.String(authenticationCode2),
		SerialNumber:        virtualMFADevice.SerialNumber,
		UserName:            aws.String(userName),
	})
	require.NoError(t, err)

	infos := &userInfos{
		UserName:  &userName,
		PolicyArn: policyOutput.Policy.Arn,
		GroupName: groupOutput.Group.GroupName,
	}

	return infos, nil
}

func deleteUserExtraResources(infos *userInfos, session *session.Session) error {
	var err error
	svc := iam.New(session)

	// Delete previously created policy
	_, err = svc.DeletePolicy(&iam.DeletePolicyInput{
		PolicyArn: infos.PolicyArn,
	})
	if err != nil {
		return err
	}

	// Delete previously created group
	_, err = svc.DeleteGroup(&iam.DeleteGroupInput{
		GroupName: infos.GroupName,
	})
	if err != nil {
		return err
	}

	return nil
}

// Validate that a user, with all the required and optional, items can be deleted
func TestDeleteFullUser(t *testing.T) {
	t.Parallel()

	region, err := getRandomRegion()
	require.NoError(t, err)

	session, err := session.NewSession(&awsgo.Config{
		Region: awsgo.String(region)},
	)
	require.NoError(t, err)

	userInfos, err := createFullTestUser(t, session)
	defer deleteUserExtraResources(userInfos, session)
	require.NoError(t, err)

	err = nukeAllIamUsers(session, []*string{userInfos.UserName})
	require.NoError(t, err)
}
