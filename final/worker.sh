# !/bin/sh

# Check if the user is root
if [ "$(id -u)" != "0" ]; then
    echo "Run this script as root/sudo"
    exit 1
fi

# Check if docker is installed
if ! [ -x "$(command -v docker)" ]; then
    echo "Docker is not installed. Please install docker first."
    exit 1
fi

# Leave docker swarm if already in swarm
sudo docker swarm leave --force > /dev/null 2>&1

# Ask for CONTROLLER_IP and PASSWORD
read -p "IP address of cluster controller: " CONTROLLER_IP
read -p "Password of cluster controller: " PASSWORD

# Send request
res=$(curl -s --fail -X POST -d "password=$PASSWORD" $CONTROLLER_IP:8080/swarm/join)

# Check if res starts with "SWMTKN"
if [ $? -ne 0 ]; then
    echo "Password is incorrect or controller is not reachable"
    exit 1
fi

echo "Joining cluster..."
# Start the worker
sudo docker swarm join --token=$res $CONTROLLER_IP:2377

# Check if worker is running
if [ $? -eq 0 ]; then
    echo "Worker is running"
else
    echo "Worker is not running"
fi