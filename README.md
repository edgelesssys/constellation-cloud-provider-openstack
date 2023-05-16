# OpenStack Cinder CSI driver for Constellation Kubernetes

This is a fork of the OpenStack Cinder CSI driver with added encryption features for Constellation.

- [Upstream source](https://github.com/kubernetes/cloud-provider-openstack)
- [Constellation repo](https://github.com/edgelesssys/constellation)

## About

This driver allows a Constellation cluster to use [Cinder CSI](https://wiki.openstack.org/wiki/Cinder) volumes, csi plugin name: `cinder.csi.confidential.cloud`

### Install the driver on a Constellation Kubernetes cluster

Create a cloud configuration:

```shell
cat <<EOF > cloud-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: cinder-csi-cloud-config
  namespace: kube-system
type: Opaque
stringData:
  cloud.conf: |-
      [Global]
      auth-url=<auth-url>
      username=<username>
      password=<password>
      project-id=<project-id>
      project-name=<project-name>
      user-domain-name=<user-domain>
      project-domain-name=<project-domain>
      region=<region>
EOF
kubectl apply -f cloud-config.yaml
```

Use `helm` to deploy the driver to your cluster:

```shell
helm install cinder-csi cloud-provider-openstack/charts/cinder-csi-plugin --namespace kube-system
```

See [helm configuration](./charts/cinder-csi-plugin/README.md) for a detailed list on configuration options.

Remove the driver using helm:

```shell
helm uninstall cinder-csi -n kube-system
```

## Features

- Please refer to [Cinder CSI Features](./docs/cinder-csi-plugin/features.md)

### Enabling integrity protection

By default the CSI driver will transparently encrypt all disks staged on the node.
Optionally, you can configure the driver to also apply integrity protection.

Please note that enabling integrity protection requires wiping the disk before use.
Disk wipe speeds are largely dependent on IOPS and the performance tier of the disk.
If you intend to provision large amounts of storage and Pod creation speed is important,
we recommend requesting high-performance disks.

To enable integrity protection, create a storage class with an explicit file system type request and add the suffix `-integrity`.
The following is a storage class for integrity protected `ext4` formatted disks:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: integrity-protected
provisioner: azuredisk.csi.confidential.cloud
parameters:
  skuName: StandardSSD_LRS
  csi.storage.k8s.io/fstype: ext4-integrity
reclaimPolicy: Delete
volumeBindingMode: Immediate
```

Please note that [volume expansion](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/) is not supported for integrity-protected disks.

## Troubleshooting

- [CSI driver troubleshooting guide](./docs/cinder-csi-plugin/troubleshooting.md)

## Kubernetes Development

- Please refer to [development guide](./docs/csi-dev.md)

To build the driver container image:

```shell
driver_version=v0.0.0-test
make REGISTRY=ghcr.io/edgelesssys/constellation VERSION=${driver_version} build-local-image-cinder-csi-plugin
docker push ghcr.io/edgelesssys/constellation/cinder-csi-plugin:${driver_version}
```

## Links

- [Kubernetes CSI Documentation](https://kubernetes-csi.github.io/docs/)
- [Container Storage Interface (CSI) Specification](https://github.com/container-storage-interface/spec)

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
