package main

import (
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

func (m Manager) addLabel(nodeIp string, label string, value string) error {
	node, err := m.client.NodeList(m.ctx, types.NodeListOptions{})
	if err != nil {
		return errors.New("failed to get node list")
	}
	var foundNode swarm.Node = swarm.Node{};
	for _, n := range node {
		if n.Status.Addr == nodeIp {
			foundNode = n;
		}
	}

	if foundNode.ID == "" {
		return errors.New("node not found")
	}

	foundNode.Spec.Labels[label] = value

	return m.client.NodeUpdate(m.ctx, foundNode.ID, foundNode.Version, foundNode.Spec)
}

func (m Manager) deleteLabel(nodeIp string, label string) error {
	node, err := m.client.NodeList(m.ctx, types.NodeListOptions{})
	if err != nil {
		return errors.New("failed to get node list")
	}	
	var foundNode swarm.Node = swarm.Node{};
	for _, n := range node {
		if n.Status.Addr == nodeIp {
			foundNode = n;
		}
	}
	
	if foundNode.ID == "" {
		return errors.New("node not found")
	}

	delete(foundNode.Spec.Labels, label)

	return m.client.NodeUpdate(m.ctx, foundNode.ID, foundNode.Version, foundNode.Spec)
}

func (m Manager) DeleteLabelsFromAllNodes(label string) error {
	node, err := m.client.NodeList(m.ctx, types.NodeListOptions{})
	if err != nil {
		return errors.New("failed to get node list")
	}	
	for _, n := range node {
		delete(n.Spec.Labels, label)
		m.client.NodeUpdate(m.ctx, n.ID, n.Version, n.Spec)
	}
	return nil
}