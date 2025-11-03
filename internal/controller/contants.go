package controller

const (
	DefaultNamespace                 = "network-policies"
	AnnotationNginxWhitelist         = "nginx.ingress.kubernetes.io/whitelist-source-range"
	AnnotationNginxDenylist          = "nginx.ingress.kubernetes.io/denylist-source-range"
	AnnotationWhiteListNetworkPolicy = "networking.k8s.io/whitelist-policy"
	AnnotationDenyListNetworkPolicy  = "networking.k8s.io/denylist-policy"
	AnnotationWhitelist              = "networking.k8s.io/whitelist"
	AnnotationDenylist               = "networking.k8s.io/denylist"
)
