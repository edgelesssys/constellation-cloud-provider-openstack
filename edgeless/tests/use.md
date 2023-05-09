# Use

## Create a new storage class

The following will create a storage class for the CSI driver.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: encrypted-storage
provisioner: cinder.csi.confidential.cloud
volumeBindingMode: WaitForFirstConsumer
EOF
```

## Make use of encrypted storage

Now you can create persistent volume claims requesting storage over your newly created storage class.
The following creates a persistent volume claim using the `encrypted-storage` class, and a Pod mounting said storage into a container:

```shell
cat <<EOF | kubectl apply -f -
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: podpvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: encrypted-storage
  resources:
    requests:
      storage: 20Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: web-server
spec:
  containers:
   - name: web-server
     image: nginx
     volumeMounts:
       - mountPath: /var/lib/www/html
         name: mypvc
  volumes:
   - name: mypvc
     persistentVolumeClaim:
       claimName: podpvc
       readOnly: false
EOF
```

## Enable integrity protection

By default the CSI driver will transparently encrypt all disks staged on the node.
Optionally, you can configure the driver to also apply integrity protection.

Please note that enabling integrity protection requires wiping the disk before use.
Disk wipe speeds are largely dependent on IOPS and the performance tier of the disk.
If you intend to provision large amounts of storage and Pod creation speed is important,
we recommend requesting high-performance disks.

To enable integrity protection, create a storage class with an explicit file system type request and add the suffix `-integrity`.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: integrity-protected
provisioner: cinder.csi.confidential.cloud
volumeBindingMode: WaitForFirstConsumer
parameters:
  csi.storage.k8s.io/fstype: ext4-integrity
```

Please note that [volume expansion](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/) is not supported for integrity-protected disks.

## [Optional] Mark the storage class as default

The default storage class is responsible for all persistent volume claims which don't explicitly request `storageClassName`.

1. List the storage classes in your cluster:

    ```shell
    kubectl get storageclass
    ```

    The output is similar to this:

    ```shell
    NAME                   PROVISIONER                        AGE
    encrypted-storage      cinder.csi.confidential.cloud      1d
    ```

    The default storage class is marked by `(default)`.

2. Mark old default storage class as non default

    If you previously used another storage class as the default, you will have to remove that annotation:

    ```shell
    kubectl patch storageclass <name-of-old-default> -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
    ```

3. Mark new class as the default

    ```shell
    kubectl patch storageclass encrypted-storage -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
    ```

4. Verify that your chosen storage class is default:

    ```shell
    kubectl get storageclass
    ```

    The output is similar to this:

    ```shell
    NAME                         PROVISIONER                        AGE
    encrypted-storage (default)  cinder.csi.confidential.cloud      1d
    ```
