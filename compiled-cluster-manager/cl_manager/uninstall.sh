# !/bin/sh

sudo systemctl stop cl_manager.service
sudo systemctl disable cl_manager.service
sudo systemctl daemon-reload
sudo rm -rf /etc/systemd/system/cl_manager.service.d
sudo rm -rf /etc/systemd/system/cl_manager.service
sudo rm -rf /usr/bin/cl_manager

echo "Run the following commands to remove docker swarm"
echo "   sudo docker swarm leave --force >&2"

echo "Uninstall complete"