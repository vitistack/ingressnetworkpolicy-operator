/*
Copyright 2025.

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

package controller

import (
	"context"
	"slices"
	"strings"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// NetworkPolicyReconciler reconciles a NetworkPolicy object
type NetworkPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies/finalizers,verbs=list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkPolicy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *NetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch NetworkPolicy that triggered this reconciliation
	var triggeredNetworkPolicy v1.NetworkPolicy
	if err := r.Get(ctx, req.NamespacedName, &triggeredNetworkPolicy); err != nil {
		log.Error(err, "unable to fetch NetworkPolicy")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Reconciling Network Policy", "NetworkPolicy.Namespace", triggeredNetworkPolicy.Namespace, "NetworkPolicy.Name", triggeredNetworkPolicy.Name)

	// Find all Ingresses
	allIngresses := &v1.IngressList{}
	matchedIngresses := []v1.Ingress{}

	if err := r.List(ctx, allIngresses); err != nil {
		log.Error(err, "unable to list Ingress")
		return ctrl.Result{}, err
	}

	// Iterate through all Ingress and find ingress that reference the NetworkPolicy
	for _, ingress := range allIngresses.Items {
		log.Info("Found Ingress", "Ingress.Name", ingress.Name)

		annotation := ingress.GetAnnotations()
		if annotation == nil {
			continue
		}

		if _, exists := annotation[AnnotationWhiteListNetworkPolicy]; exists {
			annotationList := filterSliceFromString(strings.Split(annotation[AnnotationWhiteListNetworkPolicy], ","))
			if slices.Contains(annotationList, triggeredNetworkPolicy.Name) {
				matchedIngresses = append(matchedIngresses, ingress)
			}
		}
	}

	// No Ingress matched the NetworkPolicy

	if len(matchedIngresses) == 0 {
		log.Info("No Ingress matched the NetworkPolicy", "NetworkPolicy.Name", triggeredNetworkPolicy.Name)
		return ctrl.Result{}, nil
	}

	// Update annotation on matched Ingress with CIDRs from NetworkPolicies
	for _, ingress := range matchedIngresses {

		// Declare variables
		var customWhitelist []string
		var cidrs []string

		// Get annotations from Ingress
		annotation := ingress.GetAnnotations()
		networkPolicies := filterSliceFromString(strings.Split(annotation[AnnotationWhiteListNetworkPolicy], ","))

		// Check for custom whitelist annotation
		if _, exists := annotation[AnnotationWhitelist]; exists {
			customWhitelist = filterSliceFromString(strings.Split(annotation[AnnotationWhitelist], ","))
		}

		// Get each NetworkPolicy and extract CIDRs
		for _, networkPolicy := range networkPolicies {
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

		// Append valid CIDRs from Whitelist annotation
		for _, cidr := range customWhitelist {
			if checkValidCIDR(cidr) {
				cidrs = append(cidrs, cidr)
			}
		}

		if len(cidrs) > 0 {

			// Remove duplicates and sort
			compactSortedCIDRs := sortSlice(cidrs)

			// Update Ingress annotation
			ingress.Annotations[AnnotationNginxWhitelist] = strings.Join(compactSortedCIDRs, ",")

			// Validate annotations before updating
			err := validateAnnotations(ingress.Annotations)
			if err != nil {
				log.Error(err, "invalid annotations for Ingress", "Ingress.Name", ingress.Name)
				return ctrl.Result{}, err
			}

			// Update Ingress
			if err := r.Update(ctx, &ingress); err != nil {
				log.Error(err, "unable to update Ingress annotation", "Ingress.Name", ingress.Name)
				return ctrl.Result{}, err
			}
		} else {
			// Remove NGINX annotation if no CIDRs are found
			if _, exists := ingress.Annotations[AnnotationNginxWhitelist]; exists {
				delete(ingress.Annotations, AnnotationNginxWhitelist)

				// Update Ingress
				if err := r.Update(ctx, &ingress); err != nil {
					log.Error(err, "unable to remove Ingress annotation", "Ingress.Name", ingress.Name)
					return ctrl.Result{}, err
				}
			}
		}

		// Log successful update
		log.Info("Updated Ingress annotation", "Ingress.Name", ingress.Name)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Predicate that filters updates where only annotations changed
	annotationChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectNew.GetNamespace() == DefaultNamespace
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Object.GetNamespace() == DefaultNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Object.GetNamespace() == DefaultNamespace
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.NetworkPolicy{}).
		WithEventFilter(annotationChangedPredicate).
		Named("networkpolicy").
		Complete(r)
}
