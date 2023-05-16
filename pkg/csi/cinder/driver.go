/*
Copyright (c) Edgeless Systems GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, version 3 of the License.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

This file incorporates work covered by the following copyright and
permission notice:


Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cinder

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/cloud-provider-openstack/pkg/csi/cinder/openstack"
	"k8s.io/cloud-provider-openstack/pkg/util/metadata"
	"k8s.io/cloud-provider-openstack/pkg/util/mount"
	"k8s.io/cloud-provider-openstack/pkg/version"
	"k8s.io/klog/v2"
)

const (
	driverName  = "cinder.csi.confidential.cloud"
	topologyKey = "topology." + driverName + "/zone"
)

var (
	// CSI spec version
	specVersion = "1.8.0"

	// Driver version
	// Version history:
	// * 1.3.0: Up to version 1.3.0 driver version was the same as CSI spec version
	// * 1.3.1: Bump for 1.21 release
	// * 1.3.2: Allow --cloud-config to be given multiple times
	// * 1.3.3: Bump for 1.22 release
	// * 2.0.0: Bump for 1.23 release
	Version = "2.0.0"
)

// Deprecated: use Driver instead
//
//revive:disable:exported
type CinderDriver = Driver

type cryptMapper interface {
	CloseCryptDevice(volumeID string) error
	OpenCryptDevice(ctx context.Context, source, volumeID string, integrity bool) (string, error)
	ResizeCryptDevice(ctx context.Context, volumeID string) (string, error)
	GetDevicePath(volumeID string) (string, error)
}

//revive:enable:exported

type Driver struct {
	name      string
	fqVersion string //Fully qualified version in format {Version}@{CPO version}
	endpoint  string
	cluster   string
	kmsAddr   string

	ids *identityServer
	cs  *controllerServer
	ns  *nodeServer
	cm  cryptMapper

	vcap  []*csi.VolumeCapability_AccessMode
	cscap []*csi.ControllerServiceCapability
	nscap []*csi.NodeServiceCapability
}

func NewDriver(endpoint, cluster string) *Driver {

	d := &Driver{}
	d.name = driverName
	d.fqVersion = fmt.Sprintf("%s@%s", Version, version.Version)
	d.endpoint = endpoint
	d.cluster = cluster

	klog.Info("Driver: ", d.name)
	klog.Info("Driver version: ", d.fqVersion)
	klog.Info("CSI Spec version: ", specVersion)

	d.AddControllerServiceCapabilities(
		[]csi.ControllerServiceCapability_RPC_Type{
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
			csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
			csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT,
			csi.ControllerServiceCapability_RPC_LIST_SNAPSHOTS,
			csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
			csi.ControllerServiceCapability_RPC_CLONE_VOLUME,
			csi.ControllerServiceCapability_RPC_LIST_VOLUMES_PUBLISHED_NODES,
			csi.ControllerServiceCapability_RPC_GET_VOLUME,
		})
	d.AddVolumeCapabilityAccessModes(
		[]csi.VolumeCapability_AccessMode_Mode{
			csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER,
		})

	// ignoring error, because AddNodeServiceCapabilities is public
	// and so potentially used somewhere else.
	_ = d.AddNodeServiceCapabilities(
		[]csi.NodeServiceCapability_RPC_Type{
			csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
			csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
			csi.NodeServiceCapability_RPC_GET_VOLUME_STATS,
		})

	return d
}

func (d *Driver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	csc := make([]*csi.ControllerServiceCapability, 0, len(cl))

	for _, c := range cl {
		klog.Infof("Enabling controller service capability: %v", c.String())
		csc = append(csc, NewControllerServiceCapability(c))
	}

	d.cscap = csc
}

func (d *Driver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	vca := make([]*csi.VolumeCapability_AccessMode, 0, len(vc))

	for _, c := range vc {
		klog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, NewVolumeCapabilityAccessMode(c))
	}

	d.vcap = vca

	return vca
}

func (d *Driver) AddNodeServiceCapabilities(nl []csi.NodeServiceCapability_RPC_Type) error {
	nsc := make([]*csi.NodeServiceCapability, 0, len(nl))

	for _, n := range nl {
		klog.Infof("Enabling node service capability: %v", n.String())
		nsc = append(nsc, NewNodeServiceCapability(n))
	}

	d.nscap = nsc

	return nil
}

func (d *Driver) ValidateControllerServiceRequest(c csi.ControllerServiceCapability_RPC_Type) error {
	if c == csi.ControllerServiceCapability_RPC_UNKNOWN {
		return nil
	}

	for _, cap := range d.cscap {
		if c == cap.GetRpc().GetType() {
			return nil
		}
	}

	return status.Error(codes.InvalidArgument, c.String())
}

func (d *Driver) GetVolumeCapabilityAccessModes() []*csi.VolumeCapability_AccessMode {
	return d.vcap
}

func (d *Driver) SetupDriver(cloud openstack.IOpenStack, mount mount.IMount, metadata metadata.IMetadata, cm cryptMapper) {

	d.ids = NewIdentityServer(d)
	d.cs = NewControllerServer(d, cloud)
	d.ns = NewNodeServer(d, mount, metadata, cloud)
	d.cm = cm

}

func (d *Driver) Run() {

	RunControllerandNodePublishServer(d.endpoint, d.ids, d.cs, d.ns)
}
