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
	if externalConfig == nil {
		return session.Must(
			session.NewSessionWithOptions(
				session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Config: aws.Config{
						Region: aws.String(region),
					},
				},
			),
		)
	} else {
		return session.Must(
			session.NewSessionWithOptions(
				session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Config: aws.Config{
						Region:      aws.String(region),
						Credentials: externalConfig.Credentials,
					},
				},
			),
		)
	}
}
