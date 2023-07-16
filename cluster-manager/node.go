package main

import (
	"errors"
	"sort"
	"time"

	"github.com/docker/docker/api/types"
)

var atomixNodesIP []string = make([]string, 0)
var onosNodesIP []string = make([]string, 0)

var atomixStatus map[string]bool = make(map[string]bool, 200)
var onosStatus map[string]bool = make(map[string]bool, 200)


// --- SWARM NODE ---
func (m Manager) availableNodes() ([]SwarmNode, error) {
	nodes, err := m.client.NodeList(m.ctx, types.NodeListOptions{
		
	})
	if err != nil {
		return nil, errors.New("failed to get node list")
	}
	var swarmNodes []SwarmNode
	for _, node := range nodes {
		swarmNodes = append(swarmNodes, SwarmNode{
			ID:   node.ID,
			IP:   node.Status.Addr,
			Name: node.Description.Hostname,
			Status: node.Status.State == "ready",
		})
	}
	return swarmNodes, nil
}

// --- EXISTS FUNCTIONS ---
func isAtomixNodeIPExists(ip string) bool {
	for _, nodeip := range atomixNodesIP {
		if nodeip == ip {
			return true
		}
	}
	return false
}

func isOnosNodeIPExists(ip string) bool {
	for _, nodeip := range onosNodesIP {
		if nodeip == ip {
			return true
		}
	}
	return false
}

// --- ADD FUNCTIONS ---
// --- unsafe functions, not for direct call ---
func addAtomixNode(ip string) error {
	atomixNodesIP = append(atomixNodesIP, ip)
	atomixStatus[ip] = false
	return nil
}

func addOnosNode(ip string) error {
	onosNodesIP = append(onosNodesIP, ip)
	onosStatus[ip] = false
	return nil
}

// Add ip to onosNodesIP
func addOnosNodes(m Manager, nodeips []string) error {
	for _, ip := range nodeips {
		if isOnosNodeIPExists(ip) { continue }
		if m.addLabel(ip, "onos", "1") == nil {
			onosNodesIP = append(onosNodesIP, ip)
			onosStatus[ip] = false
		}
	}
	
	sort.Strings(onosNodesIP)
	dumpOnosIPInFile()
	return nil
}

func removeOnosNodes(m Manager, nodeips []string) error {
	for _, ip := range nodeips {
		if !isOnosNodeIPExists(ip) { continue }
		if m.deleteLabel(ip, "onos") == nil {
			onosNodesIP = removeIPFromSlice(onosNodesIP, ip)
			delete(onosStatus, ip)
		}
	}
	
	sort.Strings(onosNodesIP)
	dumpOnosIPInFile()
	return nil
}

func removeIPFromSlice(slice []string, ip string) []string {
	for i, nodeip := range slice {
		if nodeip == ip {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}


// Add node to atomixNodesIP
func addAtomixNodes(m Manager, nodeips []string) error {
	if len(atomixNodesIP) > 0 {
		return errors.New("Atomix nodes already exists, reset cluster to add new nodes") 
	}
	for _, ip := range nodeips {
		if isAtomixNodeIPExists(ip) { continue }
		atomixNodesIP = append(atomixNodesIP, ip)
		atomixStatus[ip] = false
		if m.addLabel(ip, "atomix", "1") != nil {
			atomixNodesIP = make([]string, 0)
			m.DeleteLabelsFromAllNodes("atomix")
			return errors.New("failed to add label to node")
		}
	}
	
	sort.Strings(atomixNodesIP)
	dumpAtomixIPInFile()
	return nil
}



// --- ATOMIX STATUS FUNCTIONS ---
// Atomix status data structure
func upAtomix(ip string) {
	if isAtomixNodeIPExists(ip) {
		atomixStatus[ip] = true
	}
}

func downAtomix(ip string) {
	// check if ip is in atomixNodesIP
	if isAtomixNodeIPExists(ip) {
		atomixStatus[ip] = false
	}
}

func isAtomixUp(ip string) bool {
	return atomixStatus[ip]
}


// --- ONOS STATUS FUNCTIONS ---
func upOnos(ip string) {
	if isOnosNodeIPExists(ip) {
		onosStatus[ip] = true
	}
}

func downOnos(ip string) {
	if isOnosNodeIPExists(ip) {
		onosStatus[ip] = false
	}
}

func isOnosUp(ip string) bool {
	return onosStatus[ip]
}


// Config generation functions
func generateAtomixConfig(currentNode SwarmNode) string {
	if !isAtomixNodeIPExists(currentNode.IP) {
		return ""
	}
	data := make(map[string]interface{})

	//? Prepare cluster config -- START
	// Set cluster name
	data["cluster"] = make(map[string]interface{})
	data["cluster"].(map[string]interface{})["clusterId"] = "onos"

	// Set node name
	nodeDetails := make(map[string]interface{})
	nodeDetails["id"] = "atomix-"+ ConvertIPToSlug(currentNode.IP)
	// Set node address
	nodeDetails["address"] = currentNode.IP + ":5679"
	// Set node details
	data["cluster"].(map[string]interface{})["node"] = nodeDetails

	// Set discovery
	discovery := make(map[string]interface{})
	discovery["type"] = "bootstrap"
	// Set nodes
	nodes := make([]interface{}, 0)
	for _, node := range atomixNodesIP {
		nodes = append(nodes, map[string]interface{}{
			"id":     "atomix-"+ ConvertIPToSlug(node),
			"address": node + ":5679",
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].(map[string]interface{})["id"].(string) < nodes[j].(map[string]interface{})["id"].(string)
	})
	// Set nodes to discovery
	discovery["nodes"] = nodes
	// Set discovery to json
	data["cluster"].(map[string]interface{})["discovery"] = discovery

	//? Prepare cluster config -- END

	//? Prepare managementGroup config -- START
	// Set managementGroup
	managementGroup := make(map[string]interface{})
	// Set type
	managementGroup["type"] = "raft"
	// Set partitions count
	managementGroup["partitions"] = 1
	// Set members
	members := make([]string, 0)
	for _, node := range atomixNodesIP {
		members = append(members, "atomix-"+ConvertIPToSlug(node))
	}
	sort.Sort(sort.StringSlice(members))
	// Set members to managementGroup
	managementGroup["members"] = members
	// Set storage config
	storage := make(map[string]interface{})
	storage["level"] = "disk"
	// Set storage to managementGroup
	managementGroup["storage"] = storage
	// Set managementGroup to json
	data["managementGroup"] = managementGroup

	// ? Prepare managementGroup config -- END

	// ? Prepare partitionGroups config -- START
	partitionGroups := make(map[string]interface{})
	partitionGroup_raft := make(map[string]interface{})
	partitionGroup_raft["type"] = "raft"
	partitionGroup_raft["partitions"] = len(atomixNodesIP)
	partitionGroup_raft["members"] = members
	partitionGroup_raft["storage"] = storage
	partitionGroups["raft"] = partitionGroup_raft
	data["partitionGroups"] = partitionGroups

	// Convert to json
	return convertMapToJson(data)
}

func generateOnosConfig(currentNode SwarmNode) string {
	if !isOnosNodeIPExists(currentNode.IP) {
		return ""
	}
	data := make(map[string]interface{})
	// Set cluster name
	data["name"] = "onos"

	// Prepare storages
	storages := make([]interface{}, 0)
	for _, node := range atomixNodesIP {
		storages = append(storages, map[string]interface{}{
			"id": "atomix-"+ConvertIPToSlug(node),
			"ip": node,
			"port": 5679,
		})
	}

	sort.Slice(storages, func(i, j int) bool {
		return storages[i].(map[string]interface{})["id"].(string) < storages[j].(map[string]interface{})["id"].(string)
	})

	// Set storages
	data["storage"] = storages

	// Create node
	node := make(map[string]interface{})
	node["id"] = "onos-"+ ConvertIPToSlug(currentNode.IP)
	node["ip"] = currentNode.IP
	node["port"] = 9876

	// Set node
	data["node"] = node

	// Convert to json
	return convertMapToJson(data)
}

// Onos status listener
func onosStatusListener() {
	for true {
		for _, ip := range onosNodesIP {
			onosStatus[ip] = isPortOpen(ip, "6653");
		}
		time.Sleep(5 * time.Second)
	}
}
