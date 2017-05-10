package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	pkgApi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	api "k8s.io/kubernetes/pkg/api/v1"
	kubernetes "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_5"
)

func resourceKubernetesService() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesServiceCreate,
		Read:   resourceKubernetesServiceRead,
		Exists: resourceKubernetesServiceExists,
		Update: resourceKubernetesServiceUpdate,
		Delete: resourceKubernetesServiceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("service", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the behavior of a service. http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#spec-and-status",
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cluster_ip": {
							Type:        schema.TypeString,
							Description: "The IP address of the service and is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. This field can not be changed through updates. Valid values are \"None\", empty string (\"\"), or a valid IP address. \"None\" can be specified for headless services when proxying is not required. Only applies to types ClusterIP, NodePort, and LoadBalancer. Ignored if type is ExternalName. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Optional:    true,
							Computed:    true,
						},
						"external_ips": {
							Type:        schema.TypeSet,
							Description: "A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes.  The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.  A previous form of this functionality exists as the deprecatedPublicIPs field.  When using this field, callers should also clear the deprecatedPublicIPs field.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"external_name": {
							Type:        schema.TypeString,
							Description: "The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires Type to be ExternalName.",
							Optional:    true,
						},
						"load_balancer_ip": {
							Type:        schema.TypeString,
							Description: "Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.",
							Optional:    true,
						},
						"load_balancer_source_ranges": {
							Type:        schema.TypeSet,
							Description: "If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature.\" More info: http://kubernetes.io/docs/user-guide/services-firewalls",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"port": {
							Type:        schema.TypeSet,
							Description: "The list of ports that are exposed by this service. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Required:    true,
							MinItems:    1,
							Elem:        servicePortResource,
						},
						"selector": {
							Type:        schema.TypeMap,
							Description: "Route service traffic to pods with label keys and values matching this selector. If empty or not present, the service is assumed to have an external process managing its endpoints, which Kubernetes will not modify. Only applies to types ClusterIP, NodePort, and LoadBalancer. Ignored if type is ExternalName. More info: http://kubernetes.io/docs/user-guide/services#overview",
							Optional:    true,
						},
						"session_affinity": {
							Type:        schema.TypeString,
							Description: "Supports `ClientIP` and `None`. Used to maintain session affinity. Enable client IP based session affinity. Defaults to `None`. More info: http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies",
							Optional:    true,
							Default:     "None",
						},
						"type": {
							Type:        schema.TypeString,
							Description: "Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. `ClusterIP` allocates a cluster-internal IP address for load-balancing to endpoints. Endpoints are determined by the selector or if that is not specified, by manual construction of an Endpoints object. If `cluster_ip` is `None`, no virtual IP is allocated and the endpoints are published as a set of endpoints rather than a stable IP. `NodePort` builds on `cluster_ip` and allocates a port on every node which routes to the `cluster_ip`. `LoadBalancer` builds on `node_port` and creates an external load-balancer (if supported in the current cloud) which routes to the `cluster_ip`. More info: http://kubernetes.io/docs/user-guide/services#overview",
							Optional:    true,
							Default:     "ClusterIP",
						},
					},
				},
			},
		},
	}
}

var servicePortResource = &schema.Resource{
	Schema: map[string]*schema.Schema{
		"name": {
			Type:        schema.TypeString,
			Description: "The name of this port within the service. This must be a DNS_LABEL. All ports within a ServiceSpec must have unique names. This maps to the 'Name' field in EndpointPort objects. Optional if only one ServicePort is defined on this service.",
			Optional:    true,
		},
		"node_port": {
			Type:        schema.TypeInt,
			Description: "The port on each node on which this service is exposed when type=NodePort or LoadBalancer. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the ServiceType of this Service requires one. More info: http://kubernetes.io/docs/user-guide/services#type--nodeport",
			Optional:    true,
		},
		"port": {
			Type:        schema.TypeInt,
			Description: "The port that will be exposed by this service.",
			Required:    true,
		},
		"protocol": {
			Type:        schema.TypeString,
			Description: "The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.",
			Optional:    true,
			Default:     "TCP",
		},
		"target_port": {
			Type:        schema.TypeInt,
			Description: "Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME. If this is a string, it will be looked up as a named port in the target Pod's container ports. If this is not specified, the value of the 'port' field is used (an identity map). This field is ignored for services with clusterIP=None, and should be omitted or set equal to the 'port' field. More info: http://kubernetes.io/docs/user-guide/services#defining-a-service",
			Required:    true,
		},
	},
}

func resourceKubernetesServiceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	svc := api.Service{
		ObjectMeta: metadata,
		Spec:       expandServiceSpec(d.Get("spec").([]interface{})),
	}
	log.Printf("[INFO] Creating new service: %#v", svc)
	out, err := conn.CoreV1().Services(metadata.Namespace).Create(&svc)
	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceRead(d, meta)
}

func resourceKubernetesServiceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Reading service %s", name)
	svc, err := conn.CoreV1().Services(namespace).Get(name)
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}
	log.Printf("[INFO] Received service: %#v", svc)
	err = d.Set("metadata", flattenMetadata(svc.ObjectMeta))
	if err != nil {
		return err
	}

	flattened := flattenServiceSpec(svc.Spec)
	log.Printf("[DEBUG] Flattened service spec: %#v", flattened)
	err = d.Set("spec", flattened)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())

	ops := patchMetadata("metadata.0.", "/metadata/", d)
	if d.HasChange("spec") {
		diffOps := patchServiceSpec("spec.0.", "/spec/", d)
		ops = append(ops, diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating service %q: %v", name, string(data))
	out, err := conn.CoreV1().Services(namespace).Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update service: %s", err)
	}
	log.Printf("[INFO] Submitted updated service: %#v", out)
	d.SetId(buildId(out.ObjectMeta))

	return resourceKubernetesServiceRead(d, meta)
}

func resourceKubernetesServiceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Deleting service: %#v", name)
	err := conn.CoreV1().Services(namespace).Delete(name, &api.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("[INFO] Service %s deleted", name)

	d.SetId("")
	return nil
}

func resourceKubernetesServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*kubernetes.Clientset)

	namespace, name := idParts(d.Id())
	log.Printf("[INFO] Checking service %s", name)
	_, err := conn.CoreV1().Services(namespace).Get(name)
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}
