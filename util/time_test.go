package util

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

func TestParseTimestamp(t *testing.T) {
	type args struct {
		timestamp *string
	}
	tests := []struct {
		name    string
		args    args
		want    *time.Time
		wantErr bool
	}{
		{
			"it should parse legacy firstSeenTag value \"2023-12-19 10:38:44\" correctly",
			args{timestamp: aws.String("2023-12-19 10:38:44")},
			newTime(time.Date(2023, 12, 19, 10, 38, 44, 0, time.UTC)),
			false,
		},
		{
			"it should parse RFC3339 firstSeenTag value \"2024-04-12T15:18:05Z\" correctly",
			args{timestamp: aws.String("2024-04-12T15:18:05Z")},
			newTime(time.Date(2024, 4, 12, 15, 18, 5, 0, time.UTC)),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimestamp(tt.args.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseTimestamp() = %v, want %v", got, &tt.want)
			}
		})
	}
}

func newTime(t time.Time) *time.Time {
	return &t
}
