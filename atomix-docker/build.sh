# !/bin/sh

# Build the project
go build -ldflags="-s -w" -o agent 

# Build the docker image
docker build . -t atomix:custom