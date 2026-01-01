package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/gruntwork-io/cloud-nuke/config"
	"github.com/gruntwork-io/cloud-nuke/resource"
	"github.com/gruntwork-io/cloud-nuke/util"
)

// AMIsAPI defines the interface for AMI operations.
type AMIsAPI interface {
	DeregisterImage(ctx context.Context, params *ec2.DeregisterImageInput, optFns ...func(*ec2.Options)) (*ec2.DeregisterImageOutput, error)
	DescribeImages(ctx context.Context, params *ec2.DescribeImagesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeImagesOutput, error)
}

// NewAMIs creates a new AMIs resource using the generic resource pattern.
func NewAMIs() AwsResource {
	return NewAwsResource(&resource.Resource[AMIsAPI]{
		ResourceTypeName: "ami",
		BatchSize:        49,
		InitClient: WrapAwsInitClient(func(r *resource.Resource[AMIsAPI], cfg aws.Config) {
			r.Scope.Region = cfg.Region
			r.Client = ec2.NewFromConfig(cfg)
		}),
		ConfigGetter: func(c config.Config) config.ResourceType {
			return c.AMI
		},
		Lister: listAMIs,
		Nuker:  resource.SimpleBatchDeleter(deleteAMI),
	})
}

// listAMIs retrieves all user-owned AMIs that match the config filters.
func listAMIs(ctx context.Context, client AMIsAPI, scope resource.Scope, cfg config.ResourceType) ([]*string, error) {
	var imageIds []*string
	paginator := ec2.NewDescribeImagesPaginator(client, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, image := range page.Images {
			createdTime, errTimeParse := util.ParseTimestamp(image.CreationDate)
			if errTimeParse != nil {
				return nil, errTimeParse
			}

			// Check if the image has a tag that indicates AWS management
			isAWSManaged := false
			for _, tag := range image.Tags {
				if *tag.Key == "aws-managed" && *tag.Value == "true" {
					isAWSManaged = true
					break
				}
			}

			// Skip AWS managed images and images created by AWS Backup
			if isAWSManaged || strings.HasPrefix(*image.Name, "AwsBackup") {
				continue
			}

			if cfg.ShouldInclude(config.ResourceValue{
				Name: image.Name,
				Time: createdTime,
			}) {
				imageIds = append(imageIds, image.ImageId)
			}
		}
	}

	return imageIds, nil
}

// deleteAMI deletes a single AMI.
func deleteAMI(ctx context.Context, client AMIsAPI, imageID *string) error {
	_, err := client.DeregisterImage(ctx, &ec2.DeregisterImageInput{
		ImageId: imageID,
	})
	return err
}

// ImageAvailableError is returned when an image doesn't become available within wait attempts.
type ImageAvailableError struct{}

func (e ImageAvailableError) Error() string {
	return "Image didn't become available within wait attempts"
}
