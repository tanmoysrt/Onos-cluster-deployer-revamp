package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
)

// Service Spec
func (m Manager) PrepareServiceSpec(name string,
	image string,
	labels map[string]string,
	nodelabel string,
	volumes []VolumeMount,
	envs []string,
	ports []Port,
	replicas uint64,
	isGlobal bool,
	useDnsRR bool,
) swarm.ServiceSpec {
	// Create volume mounts from volume mounts array
	volumeMounts := []mount.Mount{}

	for _, volumeMount := range volumes {
		volumeMounts = append(volumeMounts, mount.Mount{
			Type:     mount.Type(volumeMount.Type),
			Source:   volumeMount.Source,
			Target:   volumeMount.Target,
			ReadOnly: volumeMount.ReadOnly,
		})
	}

	// Create ports
	portConfigs := []swarm.PortConfig{}
	for _, port := range ports {
		portConfigs = append(portConfigs, swarm.PortConfig{
			Protocol:      swarm.PortConfigProtocol(port.Protocol),
			TargetPort:    uint32(port.TargetPort),
			PublishedPort: uint32(port.PublishedPort),
			PublishMode:   swarm.PortConfigPublishMode(port.PublishedMode),
		})
	}

	var mode swarm.ServiceMode
	if isGlobal {
		mode = swarm.ServiceMode{
			Global: &swarm.GlobalService{},
		}
	} else {
		mode = swarm.ServiceMode{
			Replicated: &swarm.ReplicatedService{
				Replicas: &replicas,
			},
		}
	}

	rrMode := swarm.ResolutionModeVIP
	if useDnsRR {
		rrMode = swarm.ResolutionModeDNSRR
	}

	// Build service spec
	serviceSpec := swarm.ServiceSpec{
		// Set name of the service
		Annotations: swarm.Annotations{
			Name: name,
		},
		// Set task template
		TaskTemplate: swarm.TaskSpec{
			// Set container spec
			ContainerSpec: &swarm.ContainerSpec{
				Image:  image,
				Env:    envs,
				Mounts: volumeMounts,
				Labels: map[string]string{},
			},
			Placement: &swarm.Placement{
				Constraints: []string{
					"node.labels." + nodelabel + "==1",
				},
			},
		},
		// allow replicated service
		Mode: mode,
		// constant endpoint
		EndpointSpec: &swarm.EndpointSpec{
			Mode:  rrMode,
			Ports: portConfigs,
		},
	}
	return serviceSpec
}

// Create a new service
func (m Manager) CreateService(service swarm.ServiceSpec) error {
	_, err := m.client.ServiceCreate(m.ctx, service, types.ServiceCreateOptions{})
	if err != nil {
		return errors.New("error creating service")
	}
	return nil
}

// Update a service
func (m Manager) UpdateService(service swarm.ServiceSpec) error {
	serviceData, _, err := m.client.ServiceInspectWithRaw(m.ctx, service.Name, types.ServiceInspectOptions{})
	if err != nil {
		return errors.New("error getting swarm server version")
	}
	version := swarm.Version{
		Index: serviceData.Version.Index,
	}
	if err != nil {
		return errors.New("error getting swarm server version")
	}
	_, err = m.client.ServiceUpdate(m.ctx, service.Name, version, service, types.ServiceUpdateOptions{})
	if err != nil {
		return errors.New("error updating service")
	}
	return nil
}

// Is service exists
func (m Manager) IsServiceExists(name string) bool {
	_, _, err := m.client.ServiceInspectWithRaw(m.ctx, name, types.ServiceInspectOptions{})
	return err == nil
}

// Delete a service
func (m Manager) DeleteService(name string) error {
	err := m.client.ServiceRemove(m.ctx, name)
	if err != nil {
		return errors.New("error deleting service")
	}
	return nil
}

// Fetch nodes ip
func (m Manager) GetNodesIP() ([]string, error) {
	nodes, err := m.client.NodeList(m.ctx, types.NodeListOptions{})
	if err != nil {
		return nil, errors.New("error getting nodes")
	}
	ips := []string{}
	for _, node := range nodes {
		ips = append(ips, node.Status.Addr)
	}
	return ips, nil
}

// Fetch nodes ip
func (m Manager) getNodeDetailsFromTaskID(id string) (SwarmNode, error) {
	tasks, error := m.client.TaskList(m.ctx, types.TaskListOptions{})

	if error != nil {
		return SwarmNode{}, errors.New("failed to get container details")
	}

	var task swarm.Task
	isFound := false
	for _, t := range tasks {
		if t.Status.ContainerStatus != nil && strings.HasPrefix(t.Status.ContainerStatus.ContainerID, id) {
			task = t
			isFound = true
			break
		}
	}

	if !isFound {
		return SwarmNode{}, errors.New("failed to get node details")
	}

	node, _, err := m.client.NodeInspectWithRaw(m.ctx, task.NodeID)
	if err != nil {
		return SwarmNode{}, errors.New("failed to get node details")
	}
	return SwarmNode{
		ID:   node.ID,
		Name: node.Description.Hostname,
		IP:   node.Status.Addr,
	}, nil
}


func isPortOpen(ip string, port string) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", ip, port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Converter functions
func ConvertIPToSlug(ip string) string {
	return strings.Replace(ip, ".", "-", -1)
}

func convertMapToJson(data map[string]interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}
