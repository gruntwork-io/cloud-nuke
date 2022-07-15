package externalcreds

import (
	"github.com/aws/aws-sdk-go/aws"
)

var externalConfig *aws.Config

func Set(opts *aws.Config) {
	externalConfig = opts
}

func Get() *aws.Config {
	return externalConfig
}
