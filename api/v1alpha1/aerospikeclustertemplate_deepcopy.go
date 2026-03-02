/*
Copyright 2026.

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

package v1alpha1

// DeepCopyInto copies TemplateAerospikeConfig into out.
func (in *TemplateAerospikeConfig) DeepCopyInto(out *TemplateAerospikeConfig) {
	*out = *in
	if in.NamespaceDefaults != nil {
		out.NamespaceDefaults = in.NamespaceDefaults.DeepCopy()
	}
	if in.Service != nil {
		out.Service = in.Service.DeepCopy()
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(TemplateNetworkConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy returns a deep copy of TemplateAerospikeConfig.
func (in *TemplateAerospikeConfig) DeepCopy() *TemplateAerospikeConfig {
	if in == nil {
		return nil
	}
	out := new(TemplateAerospikeConfig)
	in.DeepCopyInto(out)
	return out
}
