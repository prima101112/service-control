# Service Control Operator

A Kubernetes operator for simplifying application deployments in Kubernetes clusters.

## Overview

The Service Control Operator is a Kubernetes controller that provides a simplified, declarative way to deploy and manage applications in a Kubernetes cluster. It introduces a custom resource called `Application` (as per Problem 4) that simplified the complexity of managing individual Kubernetes resources like Deployments, Services, Ingress, and HorizontalPodAutoscalers.

### See it in action

[![asciicast](https://asciinema.org/a/qZXm15fdk1ywq0SWzNwLE4zFn.svg)](https://asciinema.org/a/qZXm15fdk1ywq0SWzNwLE4zFn)

### Builders

using operator SDK as per the most known to build operators. and in par with kubebuilder.

### Why Service Control?

Managing microservices in Kubernetes typically requires creating and configuring multiple resources:
- Deployments for application containers
- Services for network access
- Ingress for external routing
- HPAs for auto-scaling

This often leads to repetitive YAML configurations and a steep learning curve for developers who just want to deploy their applications. Service Control solves this by:

1. **Simplifying deployments** - Define your application with a single resource
2. **Providing sensible defaults** - Common configurations are pre-set with reasonable defaults
3. **Offering customization** - Still allows full control over all underlying resources when needed, some will be defaults.
4. **Standardizing service deployments** - Enforces organization-wide deployment standards

### Why not helm or KRO

operator sdk golang provides more intuituitive development we freely added anything in more larger usecase add metrics or monitoring. also this is what i should build.

## Installation

### Prerequisites

- Kubernetes cluster 1.19+
- kubectl configured to communicate with your cluster
- Go 1.23+ (for development)

### Installing the CRD and Controller development mode / from source

1. Clone the repository:

```bash
git clone https://github.com/prima101112/service-control.git
cd service-control
```

2. Install the CRD:

```bash
make install
```

3. Deploy the controller:

```bash
make deploy
```

### Installation with Helm

You can also install the Service Control Operator using Helm:

#### Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

#### Installing the Chart

```bash
# Install the chart with the release name "service-control"
helm install service-control ./charts/service-control
```
#### Uninstalling the Chart

```bash
helm uninstall service-control
```

## Usage

### Example Applications

#### Simple Application

```yaml
apiVersion: app.teraskula.com/v1alpha1
kind: Application
metadata:
  name: simple-web
  namespace: default
spec:
  # Using jfxs/hello-world as a simple web server
  image: jfxs/hello-world
  tag: latest
  # Set 2 replicas
  replicas: 2
  # Specify the host for ingress
  host: simple-web.example.com
```

#### Advance Application Configuration

```yaml
apiVersion: app.teraskula.com/v1alpha1
kind: Application
metadata:
  name: advanced-web
  namespace: default
spec:
  # Using jfxs/hello-world as web example UI to see if its working
  image: jfxs/hello-world
  tag: latest
  # Use standard HTTP port
  port: 8080
  # Set initial replicas
  replicas: 3
  
  # Environment variables
  env:
    - name: APP_NAME
      value: "Advanced Hello World"
    - name: LOG_LEVEL
      value: "info"
    - name: CONFIG_PATH
      valueFrom:
        configMapKeyRef:
          name: hello-world-config
          key: config-path

  # Resource requests and limits
  resources:
    requests:
      memory: "64Mi"
      cpu: "100m"
    limits:
      memory: "128Mi"
      cpu: "200m"
  
  # Service configuration
  service:
    type: NodePort
    port: 80
    nodePort: 30080
    annotations:
      service.beta.kubernetes.io/description: "Hello World web service"
  
  # Ingress configuration
  ingress:
    host: advanced-web.example.com
    path: /
    tlsSecretName: advanced-web-tls
    ingressClassName: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
      nginx.ingress.kubernetes.io/rewrite-target: /
  
  # Horizontal Pod Autoscaler configuration
  # replicas will be ignored and will be using the HPA conf
  hpa:
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
  
  # Health check configuration
  healthCheck:
    path: /health
    port: 8080
    initialDelaySeconds: 30
    periodSeconds: 15
```

### Viewing Application Status

```bash
kubectl get applications
```

For detailed status:
```bash
kubectl describe application simple-web
```

## Development

Project Structure Scafold By Operator SDK

### Building from Source (docker ready and deploy to kubernetes)

```bash
# Build the controller
make docker-build docker-push 

# Run the controller locally
make deploy
```

### Building Helm Chart

```bash
# Update the Helm chart with the latest CRDs
make build-helm

```



## Resource to build this project

https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/


## My plan for next improvement

All operator metrics are already exposed in the helmchart but i am not setting any of it yet. it will be good if we have monitoring for self managed or self developed operator

https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/

## License

This project is free for anything.