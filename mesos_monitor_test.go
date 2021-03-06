package alice_test

import (
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/andygrunwald/megos"
	"github.com/notonthehighstreet/alice"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMesosClient struct {
	mock.Mock
}

func (m *MockMesosClient) GetStateFromLeader() (*megos.State, error) {
	args := m.Mock.Called()
	state := args.Get(0).(megos.State)
	return &state, args.Error(1)
}

func (m *MockMesosClient) DetermineLeader() (*megos.Pid, error) {
	args := m.Mock.Called()
	state := args.Get(0).(megos.Pid)
	return &state, args.Error(1)
}

var mockMesosClient MockMesosClient
var state megos.State
var mesosMon *alice.MesosMonitor

func setupMesosMonitorTest() {
	log = logrus.WithFields(logrus.Fields{
		"manager": "Mock",
		"monitor": "MesosMonitor",
	})
	state.Slaves = []megos.Slave{
		{
			UnreservedResources: megos.Resources{CPUs: 1.0, Disk: 1.1, Mem: 1.2},
			UsedResources:       megos.Resources{CPUs: 0.5, Disk: 0.0, Mem: 1.1},
		},
		{
			UnreservedResources: megos.Resources{CPUs: 2.0, Disk: 2.1, Mem: 2.2},
			UsedResources:       megos.Resources{CPUs: 0.5, Disk: 0.0, Mem: 1.1},
		},
		{
			UnreservedResources: megos.Resources{CPUs: 3.0, Disk: 3.1, Mem: 3.2},
			UsedResources:       megos.Resources{CPUs: 0.5, Disk: 0.0, Mem: 1.1},
		},
	}
	mockMesosClient.On("GetStateFromLeader").Return(state, nil)
	mockMesosClient.On("DetermineLeader").Return(megos.Pid{}, nil)
	m, _ := alice.NewMesosMonitor(viper.New(), log)
	mesosMon = m.(*alice.MesosMonitor)
	mesosMon.Client = &mockMesosClient
}

func TestMesosMonitor_CalculatesStatistics(t *testing.T) {
	setupMesosMonitorTest()
	stats, err := mesosMon.Stats()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, float64(25), stats.Metrics["mesos.cluster.cpu_percent"])
	assert.Equal(t, float64(50), stats.Metrics["mesos.cluster.mem_percent"])
	assert.Equal(t, float64(3.3), stats.Metrics["mesos.cluster.mem_free"])
	assert.Equal(t, float64(3.3), stats.Metrics["mesos.cluster.mem_used"])

	assert.Equal(t, float64(50), stats.Metrics["mesos.slave.cpu_percent.max"])
	assert.Equal(t, float64(34.375), stats.Metrics["mesos.slave.mem_percent.min"])

	assert.Equal(t, float64(2.1), stats.Metrics["mesos.slave.mem_free.max"])
}

func TestMesosMonitor_GetUpdatedMetrics(t *testing.T) {
	setupMesosMonitorTest()
	_, err := mesosMon.GetUpdatedMetrics([]string{"invalid.metric.name"})
	assert.NotNil(t, err)
}
