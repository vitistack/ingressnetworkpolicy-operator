# IngressNetworkPolicy-Operator

## Description
The whole purpose of the operator is to annotate ingress objects with ``nginx.ingress.kubernetes.io/whitelist-source-range`` with ip-addresses from applied network policies.

**Valid Annotations**:
``networkpolicies.networking.k8s.io/policy``
``networkpolicies.networking.k8s.io/whitelist``

## Getting Started

### Prerequisites
- Create namespace ``networkpolicies`` in the cluster
- Create ``networkpolicies.networking.k8s.io`` objects in namespace ``networkpolicies`` on demand!

### To Deploy on the cluster

**ArgoCD application definition**
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ingressnetworkpolicy-operator
  namespace: argocd
spec:
  project: default
  source:
    path: .
    repoURL: oci://ncr.sky.nhn.no/ghcr/vitistack/helm/ingressnetworkpolicy-operator
    targetRevision: 1.*
    helm:
      valueFiles:
          - values.yaml
  destination:
    server: "https://kubernetes.default.svc"
    namespace: ingressnetworkpolicy-system
  syncPolicy:
      automated:
          selfHeal: true
          prune: true
      syncOptions:
      - CreateNamespace=true
```

# License

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

