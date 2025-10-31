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
	"reflect"
	"strings"

	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// IngressReconciler reconciles a Ingress object
type IngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ingress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.1/pkg/reconcile
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Reconciling Ingress", "Ingress.Namespace", req.Namespace, "Ingress.Name", req.Name)

	// Fetch the NetworkPolicy that triggered this reconciliation
	var ingress v1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ingress); err != nil {
		log.Error(err, "unable to fetch Ingress")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get Annotations from Ingress
	annotationNetworkPolicy := ingress.GetAnnotations()[AnnotationNetworkPolicy]
	annotationWhitelist := ingress.GetAnnotations()[AnnotationWhitelist]

	// Create slices from annotations
	sliceNetworkPolicy := filterSliceFromString(strings.Split(annotationNetworkPolicy, ","))
	sliceWhitelist := filterSliceFromString(strings.Split(annotationWhitelist, ","))

	// Get each NetworkPolicy and extract CIDRs
	var cidrs []string
	for _, networkPolicy := range sliceNetworkPolicy {

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
	for _, cidr := range sliceWhitelist {
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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// Predicate that filters updates where only annotations changed
	annotationChangedPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {

			oldObjAnnotationNetworkPolicy := e.ObjectOld.GetAnnotations()[AnnotationNetworkPolicy]
			newObjAnnotationNetworkPolicy := e.ObjectNew.GetAnnotations()[AnnotationNetworkPolicy]
			oldObjAnnotationWhitelist := e.ObjectOld.GetAnnotations()[AnnotationWhitelist]
			newObjAnnotationWhitelist := e.ObjectNew.GetAnnotations()[AnnotationWhitelist]

			// Trigger reconciliation if relevant annotations have changed
			return !reflect.DeepEqual(oldObjAnnotationNetworkPolicy, newObjAnnotationNetworkPolicy) ||
				!reflect.DeepEqual(oldObjAnnotationWhitelist, newObjAnnotationWhitelist)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			// Trigger reconciliation if relevant annotations are present
			return e.Object.GetAnnotations()[AnnotationNetworkPolicy] != "" ||
				e.Object.GetAnnotations()[AnnotationWhitelist] != ""
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Ingress{}).
		Named("ingress").
		WithEventFilter(annotationChangedPredicate).
		Complete(r)
}
