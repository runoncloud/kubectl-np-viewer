# kubectl-np-viewer kubectl

A `kubectl` plugin to visualize network policies rules.

## Demo

## Examples

## Installation
There are several ways to install `np-viewer`. The recommended installation method is via `krew`.

### Via krew
Krew is a `kubectl` plugin manager. If you have not yet installed `krew`, get it at
[https://github.com/kubernetes-sigs/krew](https://github.com/kubernetes-sigs/krew).
Then installation is as simple as
```bash
kubectl krew install np-viewer
```
The plugin will be available as `kubectl np-viewer`, see [doc/USAGE](doc/USAGE.md) for further details.

### From source
Requirements:
 - go 1.14 or newer
 - git
 - GNU make
 
Compiling:
```bash
curl -L -O https://github.com/guessi/kubectl-grep/releases/download/v1.0.1/kubectl-grep-$(uname -s)-$(uname -m).tar.gz
tar zxvf kubectl-grep-$(uname -s)-$(uname -m).tar.gz
mv kubectl-np-viewer /usr/local/bin
chmod +x /usr/local/bin/kubectl-np-viewer
```
