# CSI driver e2e tests

Run CSI e2e tests using [`sonobuoy`](https://github.com/vmware-tanzu/sonobuoy/releases/latest).

## Generate test framework

Generate CSI e2e test sonobuoy config:

```shell
KUBECONFIG=</path/to/kubeconfig`
sonobuoy gen --e2e-focus='External.Storage' --e2e-skip='\[Disruptive\]' --kubeconfig=${KUBECONFIG} > sonobuoy.yaml
```

Apply driver patch:

```shell
patch sonobuoy.yaml < patch.diff
```

## Running the test suite

Start the test:

```shell
kubectl apply -f sonobuoy.yaml
```

Wait for tests to complete:

```shell
sonobuoy wait
```

Analyze results:

```shell
sonobuoy results $(sonobuoy retrieve)
```
