# kubectl np-viewer

A `kubectl` plugin to visualize network policies rules.

## Demo

<p align="center"><img src="/doc/np-viewer.gif?raw=true"/></p>

## Examples

- Prints all network policies rules for the current namespace
  ```bash
  kubectl np-viewer
  ```

- Prints all network policies rules for a specific namespace
  ```bash
  kubectl np-viewer -n default
  ```
  
- Prints all network policies rules for all namespaces
  ```bash
  kubectl np-viewer --all-namespaces
  kubectl np-viewer -A
  ```
  
- Prints network policies rules of type ingress for the current namespace
  ```bash
  kubectl np-viewer -i
  ```

- Prints network policies rules of type egress for the current namespace
  ```bash
  kubectl np-viewer -e
  ```
  
- Prints network policies rules affecting a specific pod in the current namespace
  ```bash
  kubectl np-viewer -p pod-name
  ```

## Installation
There are several ways to install `np-viewer`. The recommended installation is via the `kubectl` plugin manager 
called [`krew`](https://github.com/kubernetes-sigs/krew).

### Via krew
Krew is a `kubectl` plugin manager. If you have not yet installed `krew`, get it at
[https://github.com/kubernetes-sigs/krew](https://github.com/kubernetes-sigs/krew).
Then installation is as simple as
```bash
kubectl krew install np-viewer
```
The plugin will be available as `kubectl np-viewer`, see [doc/USAGE](doc/USAGE.md) for further details.

### Binaries
 
#### OSX
 ```bash
 curl -L -o kubectl-np-viewer.gz https://github.com/runoncloud/kubectl-np-viewer/releases/download/v1.0.6/kubectl-np-viewer_darwin_amd64.tar.gz && \
   tar zxvf kubectl-np-viewer.gz && chmod +x kubectl-np-viewer && mv kubectl-np_viewer $GOPATH/bin/
 ```
 
#### Linux
 ```bash
 curl -L -o kubectl-np-viewer.gz https://github.com/runoncloud/kubectl-np-viewer/releases/download/v1.0.6/kubectl-np-viewer_linux_amd64.tar.gz && \
   gunzip kubectl-np-viewer.gz && chmod +x kubectl-np-viewer && mv kubectl-np_viewer $GOPATH/bin/
 ```

#### Windows

 ```
 https://github.com/runoncloud/kubectl-np-viewer/releases/download/v1.0.6/kubectl-np-viewer_windows_amd64.zip
 ```

Note that the file name in the GOPATH should be `kubectl-np_viewer` if you want to be able to use it with `kubectl`

### From source

Requirements:
 - go 1.13 or newer
 - GNU make
 - git
 
 ```bash
 make bin           # binaries will be placed in bin/
 ```
