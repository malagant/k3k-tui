package types

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeepCopy creates a deep copy of the Cluster
func (c *Cluster) DeepCopy() *Cluster {
	if c == nil {
		return nil
	}

	out := &Cluster{}
	c.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies the receiver, writing into out
func (c *Cluster) DeepCopyInto(out *Cluster) {
	*out = *c
	out.TypeMeta = c.TypeMeta
	c.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	c.Spec.DeepCopyInto(&out.Spec)
	c.Status.DeepCopyInto(&out.Status)
}

// DeepCopyInto copies the receiver, writing into out for ClusterSpec
func (cs *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *cs
	
	if cs.Servers != nil {
		out.Servers = new(int32)
		*out.Servers = *cs.Servers
	}
	
	if cs.Agents != nil {
		out.Agents = new(int32)
		*out.Agents = *cs.Agents
	}
	
	if cs.Persistence != nil {
		out.Persistence = &PersistenceConfig{}
		*out.Persistence = *cs.Persistence
	}
	
	if cs.Expose != nil {
		out.Expose = &ExposeConfig{}
		if cs.Expose.Ingress != nil {
			out.Expose.Ingress = cs.Expose.Ingress.DeepCopy()
		}
		if cs.Expose.LoadBalancer != nil {
			out.Expose.LoadBalancer = cs.Expose.LoadBalancer.DeepCopy()
		}
		if cs.Expose.NodePort != nil {
			out.Expose.NodePort = cs.Expose.NodePort.DeepCopy()
		}
	}
	
	if cs.NodeSelector != nil {
		out.NodeSelector = make(map[string]string)
		for key, val := range cs.NodeSelector {
			out.NodeSelector[key] = val
		}
	}
	
	if cs.TLSSANs != nil {
		out.TLSSANs = make([]string, len(cs.TLSSANs))
		copy(out.TLSSANs, cs.TLSSANs)
	}
	
	if cs.ServerArgs != nil {
		out.ServerArgs = make([]string, len(cs.ServerArgs))
		copy(out.ServerArgs, cs.ServerArgs)
	}
	
	if cs.AgentArgs != nil {
		out.AgentArgs = make([]string, len(cs.AgentArgs))
		copy(out.AgentArgs, cs.AgentArgs)
	}
	
	if cs.ServerLimit != nil {
		out.ServerLimit = make(corev1.ResourceList)
		for key, val := range cs.ServerLimit {
			out.ServerLimit[key] = val.DeepCopy()
		}
	}
	
	if cs.WorkerLimit != nil {
		out.WorkerLimit = make(corev1.ResourceList)
		for key, val := range cs.WorkerLimit {
			out.WorkerLimit[key] = val.DeepCopy()
		}
	}
}

// DeepCopyInto copies the receiver, writing into out for ClusterStatus
func (cs *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *cs
	
	if cs.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(cs.Conditions))
		for i := range cs.Conditions {
			cs.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
}