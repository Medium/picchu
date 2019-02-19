// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterAWSInfo) DeepCopyInto(out *ClusterAWSInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterAWSInfo.
func (in *ClusterAWSInfo) DeepCopy() *ClusterAWSInfo {
	if in == nil {
		return nil
	}
	out := new(ClusterAWSInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterConditionStatus) DeepCopyInto(out *ClusterConditionStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterConditionStatus.
func (in *ClusterConditionStatus) DeepCopy() *ClusterConditionStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterConditionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterConfig) DeepCopyInto(out *ClusterConfig) {
	*out = *in
	if in.CertificateAuthorityData != nil {
		in, out := &in.CertificateAuthorityData, &out.CertificateAuthorityData
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterConfig.
func (in *ClusterConfig) DeepCopy() *ClusterConfig {
	if in == nil {
		return nil
	}
	out := new(ClusterConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterKubernetesStatus) DeepCopyInto(out *ClusterKubernetesStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterKubernetesStatus.
func (in *ClusterKubernetesStatus) DeepCopy() *ClusterKubernetesStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterKubernetesStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterList) DeepCopyInto(out *ClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterList.
func (in *ClusterList) DeepCopy() *ClusterList {
	if in == nil {
		return nil
	}
	out := new(ClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(ClusterConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.AWS != nil {
		in, out := &in.AWS, &out.AWS
		*out = new(ClusterAWSInfo)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSpec.
func (in *ClusterSpec) DeepCopy() *ClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *in
	out.Kubernetes = in.Kubernetes
	if in.AWS != nil {
		in, out := &in.AWS, &out.AWS
		*out = new(ClusterAWSInfo)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]ClusterConditionStatus, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatus.
func (in *ClusterStatus) DeepCopy() *ClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Incarnation) DeepCopyInto(out *Incarnation) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Incarnation.
func (in *Incarnation) DeepCopy() *Incarnation {
	if in == nil {
		return nil
	}
	out := new(Incarnation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Incarnation) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationApp) DeepCopyInto(out *IncarnationApp) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationApp.
func (in *IncarnationApp) DeepCopy() *IncarnationApp {
	if in == nil {
		return nil
	}
	out := new(IncarnationApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationAssignment) DeepCopyInto(out *IncarnationAssignment) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationAssignment.
func (in *IncarnationAssignment) DeepCopy() *IncarnationAssignment {
	if in == nil {
		return nil
	}
	out := new(IncarnationAssignment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationHealthMetricStatus) DeepCopyInto(out *IncarnationHealthMetricStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationHealthMetricStatus.
func (in *IncarnationHealthMetricStatus) DeepCopy() *IncarnationHealthMetricStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationHealthMetricStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationHealthStatus) DeepCopyInto(out *IncarnationHealthStatus) {
	*out = *in
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = make([]IncarnationHealthMetricStatus, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationHealthStatus.
func (in *IncarnationHealthStatus) DeepCopy() *IncarnationHealthStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationHealthStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationList) DeepCopyInto(out *IncarnationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Incarnation, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationList.
func (in *IncarnationList) DeepCopy() *IncarnationList {
	if in == nil {
		return nil
	}
	out := new(IncarnationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IncarnationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationRelease) DeepCopyInto(out *IncarnationRelease) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationRelease.
func (in *IncarnationRelease) DeepCopy() *IncarnationRelease {
	if in == nil {
		return nil
	}
	out := new(IncarnationRelease)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationReleaseStatus) DeepCopyInto(out *IncarnationReleaseStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationReleaseStatus.
func (in *IncarnationReleaseStatus) DeepCopy() *IncarnationReleaseStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationReleaseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationResourceStatus) DeepCopyInto(out *IncarnationResourceStatus) {
	*out = *in
	out.Metadata = in.Metadata
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationResourceStatus.
func (in *IncarnationResourceStatus) DeepCopy() *IncarnationResourceStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationResourceStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationScale) DeepCopyInto(out *IncarnationScale) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]IncarnationScaleResource, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationScale.
func (in *IncarnationScale) DeepCopy() *IncarnationScale {
	if in == nil {
		return nil
	}
	out := new(IncarnationScale)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationScaleResource) DeepCopyInto(out *IncarnationScaleResource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationScaleResource.
func (in *IncarnationScaleResource) DeepCopy() *IncarnationScaleResource {
	if in == nil {
		return nil
	}
	out := new(IncarnationScaleResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationScaleStatus) DeepCopyInto(out *IncarnationScaleStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationScaleStatus.
func (in *IncarnationScaleStatus) DeepCopy() *IncarnationScaleStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationScaleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationSpec) DeepCopyInto(out *IncarnationSpec) {
	*out = *in
	out.App = in.App
	out.Assignment = in.Assignment
	in.Scale.DeepCopyInto(&out.Scale)
	out.Release = in.Release
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortInfo, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationSpec.
func (in *IncarnationSpec) DeepCopy() *IncarnationSpec {
	if in == nil {
		return nil
	}
	out := new(IncarnationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IncarnationStatus) DeepCopyInto(out *IncarnationStatus) {
	*out = *in
	in.Health.DeepCopyInto(&out.Health)
	out.Release = in.Release
	out.Scale = in.Scale
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]IncarnationResourceStatus, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IncarnationStatus.
func (in *IncarnationStatus) DeepCopy() *IncarnationStatus {
	if in == nil {
		return nil
	}
	out := new(IncarnationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PortInfo) DeepCopyInto(out *PortInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PortInfo.
func (in *PortInfo) DeepCopy() *PortInfo {
	if in == nil {
		return nil
	}
	out := new(PortInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Revision) DeepCopyInto(out *Revision) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Revision.
func (in *Revision) DeepCopy() *Revision {
	if in == nil {
		return nil
	}
	out := new(Revision)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Revision) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionApp) DeepCopyInto(out *RevisionApp) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionApp.
func (in *RevisionApp) DeepCopy() *RevisionApp {
	if in == nil {
		return nil
	}
	out := new(RevisionApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionList) DeepCopyInto(out *RevisionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Revision, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionList.
func (in *RevisionList) DeepCopy() *RevisionList {
	if in == nil {
		return nil
	}
	out := new(RevisionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RevisionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionRelease) DeepCopyInto(out *RevisionRelease) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionRelease.
func (in *RevisionRelease) DeepCopy() *RevisionRelease {
	if in == nil {
		return nil
	}
	out := new(RevisionRelease)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionSpec) DeepCopyInto(out *RevisionSpec) {
	*out = *in
	out.App = in.App
	out.Release = in.Release
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]PortInfo, len(*in))
		copy(*out, *in)
	}
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]RevisionTarget, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionSpec.
func (in *RevisionSpec) DeepCopy() *RevisionSpec {
	if in == nil {
		return nil
	}
	out := new(RevisionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionStatus) DeepCopyInto(out *RevisionStatus) {
	*out = *in
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]RevisionTargetStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionStatus.
func (in *RevisionStatus) DeepCopy() *RevisionStatus {
	if in == nil {
		return nil
	}
	out := new(RevisionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTarget) DeepCopyInto(out *RevisionTarget) {
	*out = *in
	in.Scale.DeepCopyInto(&out.Scale)
	if in.Release != nil {
		in, out := &in.Release, &out.Release
		*out = new(RevisionTargetRelease)
		**out = **in
	}
	if in.Metrics != nil {
		in, out := &in.Metrics, &out.Metrics
		*out = make([]RevisionTargetMetric, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTarget.
func (in *RevisionTarget) DeepCopy() *RevisionTarget {
	if in == nil {
		return nil
	}
	out := new(RevisionTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetDeploymentStatus) DeepCopyInto(out *RevisionTargetDeploymentStatus) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetDeploymentStatus.
func (in *RevisionTargetDeploymentStatus) DeepCopy() *RevisionTargetDeploymentStatus {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetDeploymentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetMetric) DeepCopyInto(out *RevisionTargetMetric) {
	*out = *in
	out.Queries = in.Queries
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetMetric.
func (in *RevisionTargetMetric) DeepCopy() *RevisionTargetMetric {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetMetric)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetMetricQueries) DeepCopyInto(out *RevisionTargetMetricQueries) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetMetricQueries.
func (in *RevisionTargetMetricQueries) DeepCopy() *RevisionTargetMetricQueries {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetMetricQueries)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetRelease) DeepCopyInto(out *RevisionTargetRelease) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetRelease.
func (in *RevisionTargetRelease) DeepCopy() *RevisionTargetRelease {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetRelease)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetScale) DeepCopyInto(out *RevisionTargetScale) {
	*out = *in
	in.Resources.DeepCopyInto(&out.Resources)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetScale.
func (in *RevisionTargetScale) DeepCopy() *RevisionTargetScale {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetScale)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RevisionTargetStatus) DeepCopyInto(out *RevisionTargetStatus) {
	*out = *in
	if in.Deployments != nil {
		in, out := &in.Deployments, &out.Deployments
		*out = make([]RevisionTargetDeploymentStatus, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RevisionTargetStatus.
func (in *RevisionTargetStatus) DeepCopy() *RevisionTargetStatus {
	if in == nil {
		return nil
	}
	out := new(RevisionTargetStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScaleResources) DeepCopyInto(out *ScaleResources) {
	*out = *in
	out.CPU = in.CPU.DeepCopy()
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScaleResources.
func (in *ScaleResources) DeepCopy() *ScaleResources {
	if in == nil {
		return nil
	}
	out := new(ScaleResources)
	in.DeepCopyInto(out)
	return out
}
