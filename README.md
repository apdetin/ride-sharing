# Ride Sharing

The backend microservices system for a Uber‑style ride‑sharing app from the ground up—using Go, Docker, and Kubernetes.

## Prerequisites

- go 1.23
- minikube
- tilt
- docker
- kubectl
- https://httpyac.github.io for testing

## Logs

- create new service using `go run tools/create_service.go -name <service-name>`
- ensure minikube is running using `kubectl config current-context` and `kubectl cluster-info`
- run `tilt up` to start the development server