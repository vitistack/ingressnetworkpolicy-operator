package controller

import (
	"context"

	v1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func createCidrList(ctx context.Context, ingress v1.Ingress, policyList []string, customList []string) []string {
	log := logf.FromContext(ctx)

	// Create Kubernetes client
	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Error(err, "unable to get config")
	}

	c, err := client.New(cfg, client.Options{})
	if err != nil {
		log.Error(err, "unable to create client")
	}

	var cidrs []string

	// Get each NetworkPolicy and extract CIDRs

	for _, networkPolicy := range policyList {

		processNetworkPolicy := v1.NetworkPolicy{}

		err = c.Get(ctx, client.ObjectKey{
			Namespace: DefaultNamespace,
			Name:      networkPolicy,
		}, &processNetworkPolicy)

		if err != nil {
			log.Error(err, "unable to fetch NetworkPolicy for Ingress", "Ingress.Name", ingress.Name, "ExpectedPolicy", networkPolicy)
			continue
		}

		// Extract CIDRs from NetworkPolicy and append to list
		cidrs = append(cidrs, extractCIDRsFromNetworkPolicy(&processNetworkPolicy, cidrs)...)
	}

	// Append valid CIDRs from customList
	for _, cidr := range customList {
		if checkValidCIDR(cidr) {
			cidrs = append(cidrs, cidr)
		}
	}

	// Remove duplicates and sort
	compactSortedCIDRs := sortSlice(cidrs)

	return compactSortedCIDRs
}
