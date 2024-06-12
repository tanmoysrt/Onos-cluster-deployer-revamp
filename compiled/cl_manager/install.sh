# !/bin/sh

# If not root, run as root
if [ "$(id -u)" -ne 0 ]
  then echo "Please run as root"
  exit
fi

# Check if docker is installed
if ! [ -x "$(command -v docker)" ]; then
  echo 'Error: docker is not installed.' >&2
  exit 1
fi


# Check if /var/run/docker.sock exists
if [ ! -S "/var/run/docker.sock" ]; then
  echo 'Error: /var/run/docker.sock does not exist.' >&2
  exit 1
fi

# Try to delete from swarm
sudo docker swarm leave --force >&2

# Take PASSWORD,CLUSTER_MANAGER_ADDRESS,ATOMIX_IMAGE,ONOS_IMAGE as input
read -p "Enter PASSWORD: " PASSWORD
read -p "Enter CLUSTER_MANAGER_ADDRESS: " CLUSTER_MANAGER_ADDRESS
read -p "Enter CLUSTER_MANAGER_IP: " CLUSTER_MANAGER_IP
read -p "Enter ATOMIX_IMAGE: " ATOMIX_IMAGE
read -p "Enter ONOS_IMAGE: " ONOS_IMAGE


# Initialize swarm
sudo docker swarm init --advertise-addr=$CLUSTER_MANAGER_IP


# Store this in /etc/systemd/system/cl_manager.service.d/local.conf
sudo mkdir -p /etc/systemd/system/cl_manager.service.d
sudo touch /etc/systemd/system/cl_manager.service.d/local.conf
sudo chmod 600 /etc/systemd/system/cl_manager.service.d/local.conf
sudo echo "[Service]" >> /etc/systemd/system/cl_manager.service.d/local.conf
sudo echo "Environment=\"PASSWORD=$PASSWORD\"" >> /etc/systemd/system/cl_manager.service.d/local.conf
sudo echo "Environment=\"CLUSTER_MANAGER_ADDRESS=$CLUSTER_MANAGER_ADDRESS\"" >> /etc/systemd/system/cl_manager.service.d/local.conf
sudo echo "Environment=\"ATOMIX_IMAGE=$ATOMIX_IMAGE\"" >> /etc/systemd/system/cl_manager.service.d/local.conf
sudo echo "Environment=\"ONOS_IMAGE=$ONOS_IMAGE\"" >> /etc/systemd/system/cl_manager.service.d/local.conf


# Move cl_manager to /usr/bin
sudo cp cl_manager /usr/bin

# Move cl_manager.service to /etc/systemd/system
sudo cp cl_manager.service /etc/systemd/system

# Make RW by root only
sudo chmod 600 /etc/systemd/system/cl_manager.service

# Daemon-reload
sudo systemctl daemon-reload

# Enable cl_manager.service
sudo systemctl enable cl_manager.service

# Start cl_manager.service
sudo systemctl start cl_manager.service

# Message
echo "cl_manager.service is started."
