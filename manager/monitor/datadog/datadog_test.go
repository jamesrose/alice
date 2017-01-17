package datadog_test

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/notonthehighstreet/autoscaler/manager/monitor/datadog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	datadogclient "github.com/zorkian/go-datadog-api"
	"testing"
	"time"
)

var log = logrus.WithFields(logrus.Fields{
	"manager": "Mock",
	"monitor": "DatadogMonitor",
})
var monitor *datadog.DatadogMonitor
var config *viper.Viper
var mockDatadogClient datadog.MockDatadogClient

func setupTest() {
	config = viper.New()
	config.Set("api_key", "foo")
	config.Set("app_key", "bar")
	config.Set("time_period", "5m")
	log.Logger.Level = logrus.DebugLevel
	mockDatadogClient = datadog.MockDatadogClient{}
	m, _ := datadog.New(config, log)
	monitor = m.(*datadog.DatadogMonitor)
	monitor.Client = &mockDatadogClient
}

func TestDatadog_GetUpdatedMetrics(t *testing.T) {
	setupTest()
	config.Set("metrics.foo.bar.baz.query", "avg:foo.bar.baz{*}")
	metrics := []string{"foo.bar.baz"}
	mockResponse := []datadogclient.Series{
		{Points: []datadogclient.DataPoint{
			{float64(time.Now().Unix() - 2), 0.9},
			{float64(time.Now().Unix() - 1), 0.3},
			{float64(time.Now().Unix()), 0.5},
		}},
	}
	mockDatadogClient.On("Validate").Return(true, nil)
	mockDatadogClient.On("QueryMetrics").Return(mockResponse, nil)
	vp, err := monitor.GetUpdatedMetrics(metrics)
	assert.NoError(t, err)
	val := *vp
	assert.Equal(t, 1, len(val))
	assert.Equal(t, 0.5, val[0].CurrentReading)
}

func TestDatadog_GetUpdatedMetricsNoData(t *testing.T) {
	setupTest()
	metrics := []string{"foo.bar.baz"}
	mockDatadogClient.On("Validate").Return(true, nil)

	mockResponse := []datadogclient.Series{}
	mockDatadogClient.On("QueryMetrics").Return(mockResponse, nil).Once()
	_, err1 := monitor.GetUpdatedMetrics(metrics)
	assert.Error(t, err1)

	mockResponse = []datadogclient.Series{
		{Points: []datadogclient.DataPoint{}},
	}
	mockDatadogClient.On("QueryMetrics").Return(mockResponse, nil).Once()
	_, err2 := monitor.GetUpdatedMetrics(metrics)
	assert.Error(t, err2)
}

func TestDatadogMonitorInvalidApiKey(t *testing.T) {
	setupTest()
	metrics := []string{"foo.bar.baz"}
	mockDatadogClient.On("Validate").Return(false, nil).Once()
	_, eA := monitor.GetUpdatedMetrics(metrics)
	assert.Error(t, eA)
	mockDatadogClient.On("Validate").Return(true, errors.New("")).Once()
	_, eB := monitor.GetUpdatedMetrics(metrics)
	assert.Error(t, eB)
}
