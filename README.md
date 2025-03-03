# must-hydrate

must-hydrate builds an envtest, or possibly any lightweight, control plane from a must-gather. The goal is to provide a fast way of consuming
a must-gather for the purposes of investigation or unit test development. For numerous activites, access to a 'live' cluster isn't always necessary
and spinning up a new cluster can be expensive and time consuming. 
![demo.gif](./docs/demo.gif)

## Type Support

Types from github.com/openshift/api and all CRDs from the must-gather are installed in to the local control plane. 

## Building

must-hydrate can be built as a container:

```sh
podman build . -t must_hydrate
```

or as a standalone binary:

```sh
make build
```

## Running

### Prerequisites

- A single extracted `must-gather` in a directory
- If running as a container(recommended), the directory which contains the must-gather must be mounted to the container as /data.
  The /data path will be recursed and all yamls found will be processed
- The kubeconfig to be used to interrogate the must-gather will be written to /data/envtest.kubeconfig if running in a container or the working
  directory if not running in a container.

### Starting must_hydrate
```sh
podman run -v $(pwd)/data:/data:z --network host must_hydrate
```

The api server is started on a random port at this time and as such must run on the host network. It is possible, however, to 
modify this to not require host networking.

### Accessing the API

```sh
$ export KUBECONFIG=./data-dir/envtest.kubeconfig
$ oc get co
NAME                                       VERSION                              AVAILABLE   PROGRESSING   DEGRADED   SINCE
authentication                             4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
baremetal                                  4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
cloud-controller-manager                   4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
cloud-credential                           4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
cluster-api                                4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
cluster-autoscaler                         4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
config-operator                            4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
console                                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
control-plane-machine-set                  4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
csi-snapshot-controller                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
dns                                        4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
etcd                                       4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
image-registry                             4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
ingress                                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
insights                                   4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
kube-apiserver                             4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
kube-controller-manager                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
kube-scheduler                             4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
kube-storage-version-migrator              4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
machine-api                                4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
machine-approver                           4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
machine-config                             4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
marketplace                                4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
monitoring                                 4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
network                                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
node-tuning                                4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
olm                                        4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
openshift-apiserver                        4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
openshift-controller-manager               4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
openshift-samples                          4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
operator-lifecycle-manager                 4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
operator-lifecycle-manager-catalog         4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
operator-lifecycle-manager-packageserver   4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d11h
service-ca                                 4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
storage                                    4.19.0-0.nightly-2025-02-14-215306   True        False         False      7d12h
```

### Using with openshift-tests

In order to perform testing with openshift-tests(i.e. you need to add a test) you will need to obtain a client that does not create a new project. For example:

```sh
	//oc := exutil.NewCLI("cluster-client-cert")
	oc := exutil.NewCLIWithoutNamespace("cluster-client-cert")
```
The reason being that oc provisions resources which require a controller to create service accounts, credentials, etc.. In theory, you could provision the necessary controllers to handle this.
However, for many use cases this workaround should be sufficient.  If you need controllers, install a cluster.

## Log File Support

Pod logs are generally retrieved via the kubelet daemon port. Since there is no kubelet, `localhost:10250` is opened and listening for log requests. In turn, the Node resources are updated to set their hostname to localhost.  `oc` uses the hostname to determine which kubelet is associated with the logs to be gatherered. Setting the hostnames to `localhost` forces them all requests through must-hydrate.
Logs can be disabled by passing `--disable-logs=true`.

## Troubleshooting

### Too many file handles

Bump up the maximum number of file handles. How and what to set this to will vary based on your specific OS/environment.
```sh
sudo sysctl -w fs.inotify.max_user_watches=2099999999
sudo sysctl -w fs.inotify.max_user_instances=2099999999
sudo sysctl -w fs.inotify.max_queued_events=2099999999
```
