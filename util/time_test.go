package util

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestParseTimestamp(t *testing.T) {
	type args struct {
		timestamp string
	}
	tests := []struct {
		name    string
		args    args
		want    *time.Time
		wantErr bool
	}{
		{
			"it should parse legacy firstSeenTag value \"2023-12-19 10:38:44\" correctly",
			args{timestamp: "2023-12-19 10:38:44"},
			newTime(time.Date(2023, 12, 19, 10, 38, 44, 0, time.UTC)),
			false,
		},
		{
			"it should parse RFC3339 firstSeenTag value \"2024-04-12T15:18:05Z\" correctly",
			args{timestamp: "2024-04-12T15:18:05Z"},
			newTime(time.Date(2024, 4, 12, 15, 18, 5, 0, time.UTC)),
			false,
		},
		{
			"it should parse bare ISO 8601 value \"2015-01-01T00:00:00\" correctly",
			args{timestamp: "2015-01-01T00:00:00"},
			newTime(time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)),
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

func TestParseTimestampPtr(t *testing.T) {
	got, err := ParseTimestampPtr(aws.String("2024-04-12T15:18:05Z"))
	if err != nil {
		t.Fatalf("ParseTimestampPtr() unexpected error: %v", err)
	}
	want := time.Date(2024, 4, 12, 15, 18, 5, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("ParseTimestampPtr() = %v, want %v", got, want)
	}

	// nil input should parse empty string and fail
	_, err = ParseTimestampPtr(nil)
	if err == nil {
		t.Error("ParseTimestampPtr(nil) expected error, got nil")
	}
}

func newTime(t time.Time) *time.Time {
	return &t
}
