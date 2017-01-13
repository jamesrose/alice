package mesos

import (
	"github.com/Sirupsen/logrus"
	"github.com/andygrunwald/megos"
	"github.com/notonthehighstreet/autoscaler/manager/monitor"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/url"
)

type MesosMonitor struct {
	log    *logrus.Entry
	Client MesosClient
	config *viper.Viper
}

type MesosStats struct {
	Metrics map[string]float64
}

func (s *MesosStats) updateMinMax(name string, number float64) {
	if min, ok := s.Metrics[name+".min"]; ok {
		if min > number {
			s.Metrics[name+".min"] = number
		}
	} else {
		s.Metrics[name+".min"] = number
	}
	if max, ok := s.Metrics[name+".max"]; ok {
		if max < number {
			s.Metrics[name+".max"] = number
		}
	} else {
		s.Metrics[name+".max"] = number
	}
}

type MesosClient interface {
	GetStateFromLeader() (*megos.State, error)
	DetermineLeader() (*megos.Pid, error)
}

const defaultMesosMaster = "http://mesos.service.consul:5050/state"

// NewMesosMaster initialises any new Mesos master. We will use this master to determine the leader of the cluster.
func New(config *viper.Viper, log *logrus.Entry) (monitor.Monitor, error) {
	config.SetDefault("endpoint", defaultMesosMaster)
	u, err := url.Parse(config.GetString("endpoint"))
	if err != nil {
		return nil, errors.Wrap(err, "Can't create mesos monitor")
	}
	mesos := megos.NewClient([]*url.URL{u}, nil)
	return &MesosMonitor{log: log, Client: mesos, config: config}, nil
}

func (m *MesosMonitor) GetUpdatedMetrics(names []string) (*[]monitor.MetricUpdate, error) {
	response := make([]monitor.MetricUpdate, len(names))
	stats, err := m.Stats()
	if err != nil {
		return nil, err
	}
	for i, name := range names {
		response[i].Name = name
		if val, ok := stats.Metrics[name]; ok {
			response[i].CurrentReading = val
		} else {
			return &response, errors.Errorf("Unknown mesos metric: %s", name)
		}
	}
	return &response, nil
}

func (m *MesosMonitor) Stats() (*MesosStats, error) {
	m.Client.DetermineLeader()
	state, err := m.Client.GetStateFromLeader()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Mesos stats")
	}

	stats := &MesosStats{}
	stats.Metrics = make(map[string]float64)

	for _, slave := range state.Slaves {
		stats.Metrics["mesos.cluster.cpu_total"] += slave.UnreservedResources.CPUs
		stats.Metrics["mesos.cluster.mem_total"] += slave.UnreservedResources.Mem
		stats.Metrics["mesos.cluster.cpu_used"] += slave.UsedResources.CPUs
		stats.Metrics["mesos.cluster.mem_used"] += slave.UsedResources.Mem

		stats.updateMinMax("mesos.slave.cpu_free", slave.UnreservedResources.CPUs-slave.UsedResources.CPUs)
		stats.updateMinMax("mesos.slave.cpu_used", slave.UsedResources.CPUs)
		stats.updateMinMax("mesos.slave.cpu_percent", (slave.UsedResources.CPUs*10)*100/(slave.UnreservedResources.CPUs*10))
		stats.updateMinMax("mesos.slave.mem_free", slave.UnreservedResources.Mem-slave.UsedResources.Mem)
		stats.updateMinMax("mesos.slave.mem_used", slave.UsedResources.Mem)
		stats.updateMinMax("mesos.slave.mem_percent", (slave.UsedResources.Mem*10)*100/(slave.UnreservedResources.Mem*10))
	}
	stats.Metrics["mesos.cluster.cpu_percent"] = (stats.Metrics["mesos.cluster.cpu_used"] * 10) * 100 / (stats.Metrics["mesos.cluster.cpu_total"] * 10)
	stats.Metrics["mesos.cluster.mem_percent"] = (stats.Metrics["mesos.cluster.mem_used"] * 10) * 100 / (stats.Metrics["mesos.cluster.mem_total"] * 10)

	return stats, nil
}
