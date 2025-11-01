package controller

import (
	"context"

	v1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func createCidrList(ctx context.Context, r *IngressReconciler, ingress v1.Ingress, policyList []string, customList []string) []string {
	log := logf.FromContext(ctx)

	var cidrs []string

	// Get each NetworkPolicy and extract CIDRs

	for _, networkPolicy := range policyList {

		processNetworkPolicy := v1.NetworkPolicy{}
		err := r.Get(ctx, client.ObjectKey{
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
