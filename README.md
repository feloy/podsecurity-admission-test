# Pod Security Admission test

This example uses the `k8s.io/pod-security-admission` library to check if a Pod passes 
the checks for the Pod Security Admission restrictions set in a given namespace.

## Usage

- Create a namespace `ns1` with restrctions:
  ```console
  $ kubectl create namespace ns1
  $ kubectl label namespaces ns1 \
      pod-security.kubernetes.io/enforce=restricted
  $ kubectl label namespaces ns1 \
      pod-security.kubernetes.io/enforce-version=v1.25
  ```
- Execute:
  ```console
  $ go run main.go
  Enforce policy: (restricted, v1.25)
  - allowPrivilegeEscalation != false
    container "nginx" must set securityContext.allowPrivilegeEscalation=false
  - unrestricted capabilities
    container "nginx" must set securityContext.capabilities.drop=["ALL"]
  - runAsNonRoot != true
    pod or container "nginx" must set securityContext.runAsNonRoot=true
  - seccompProfile
    pod or container "nginx" must set securityContext.seccompProfile.type to "RuntimeDefault" or "Localhost"
  ```
