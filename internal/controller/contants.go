package controller

const (
	DefaultNamespace         = "default"
	AnnotationNginxWhitelist = "nginx.ingress.kubernetes.io/whitelist-source-range"
	AnnotationNetworkPolicy  = "networking.k8s.io/policy"
	AnnotationWhitelist      = "networking.k8s.io/whitelist"
)
