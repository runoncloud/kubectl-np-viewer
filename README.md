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

#### Build on host

Requirements:
 - go 1.13 or newer
 - GNU make
 - git

Compiling:
```bash
export PLATFORMS=$(go env GOOS)
make all   # binaries will be placed in out/
```
