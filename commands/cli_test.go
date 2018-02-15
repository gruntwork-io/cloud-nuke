package commands

import (
	"testing"
	"time"

	"github.com/gruntwork-io/gruntwork-cli/errors"
	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {
	dateString := "01-01-2018 00:00AM"
	parsedTime, err := parseTimeParam(dateString)
	if err != nil {
		assert.Fail(t, errors.WithStackTrace(err).Error())
	}

	assert.Equal(t, parsedTime.Month(), time.Month(1))
	assert.Equal(t, parsedTime.Day(), 1)
	assert.Equal(t, parsedTime.Year(), 2018)
}

func TestParseTimeInvalidFormat(t *testing.T) {
	_, err := parseTimeParam("")
	assert.Error(t, err)

	_, err = parseTimeParam("01/01/2018")
	assert.Error(t, err)
}
