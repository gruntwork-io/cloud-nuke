package externalcreds

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var externalConfig *aws.Config

func Set(opts *aws.Config) {
	externalConfig = opts
}

func Get(region string) *session.Session {
	config := aws.Config{
		Region: aws.String(region),
	}
	// If external config was passed in, use its credentials
	if externalConfig != nil {
		config.Credentials = externalConfig.Credentials
	}
	return session.Must(
		session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Config:            config,
			},
		),
	)
}
