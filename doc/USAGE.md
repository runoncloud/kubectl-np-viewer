
## Usage
The following assumes you have the plugin installed via [krew](krew).

```shell
kubectl krew install np-viewer
```

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
