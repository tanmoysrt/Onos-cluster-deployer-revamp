package main

import (
	"os"

	"github.com/docker/docker/api/types"
)

// Onos
func (m Manager) CreateOnosService(name string, image string) error {
	onosMetdataUrl := os.Getenv("CLUSTER_MANAGER_ADDRESS") + "/onos/config/"
	onosMetdataUrlENV := "METADATA_URL=" + onosMetdataUrl
	OnosServiceSpec := m.PrepareServiceSpec(
		name,
		image,
		map[string]string{},
		"onos",
		[]VolumeMount{},
		[]string{
			onosMetdataUrlENV,
			"ONOS_APPS=drivers,gui2,openflow,fwd",
		},
		[]Port{
			{
				Protocol:      "tcp",
				TargetPort:    6653,
				PublishedPort: 6653,
				PublishedMode: "host",
			},
			{
				Protocol:      "tcp",
				TargetPort:    6640,
				PublishedPort: 6640,
				PublishedMode: "host",
			},
			{
				Protocol:      "tcp",
				TargetPort:    9876,
				PublishedPort: 9876,
				PublishedMode: "host",
			},
			{
				Protocol:      "tcp",
				TargetPort:    8181,
				PublishedPort: 8181,
				PublishedMode: "host",
			},
			{
				Protocol:      "tcp",
				TargetPort:    8101,
				PublishedPort: 8101,
				PublishedMode: "host",
			},
		},
		uint64(0),
		true,
		false,
	)
	err := m.CreateService(OnosServiceSpec)
	return err
}

// Atomix

func (m Manager) CreateAtomixService(name string, image string) error {
	clusterManagerIpENV := "CLUSTER_MANAGER_ADDRESS=" + os.Getenv("CLUSTER_MANAGER_ADDRESS")
	atomixServiceSpec := m.PrepareServiceSpec(
		name,
		image,
		map[string]string{},
		"atomix",
		[]VolumeMount{},
		[]string{
			clusterManagerIpENV,
		},
		[]Port{
			{
				Protocol:      "tcp",
				TargetPort:    5678,
				PublishedPort: 5678,
				PublishedMode: "host",
			},
			{
				Protocol:      "tcp",
				TargetPort:    5679,
				PublishedPort: 5679,
				PublishedMode: "host",
			},
		},
		uint64(0),
		true,
		false,
	)
	err := m.CreateService(atomixServiceSpec)
	return err
}

func (m Manager) ServiceExists(name string) bool {
	_, _, err := m.client.ServiceInspectWithRaw(m.ctx, name, types.ServiceInspectOptions{})
	return err == nil
}
