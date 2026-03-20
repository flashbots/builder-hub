#!/bin/sh
set -e

export VAULT_ADDR="http://vault:8200"
export VAULT_TOKEN="root-token"

# Generate RSA key pair for signing the SA JWT (RS256 required by Vault k8s auth)
openssl genrsa -out /tmp/sa.key 2048
openssl rsa -in /tmp/sa.key -pubout -out /tmp/sa.pub

# Create a k8s-style SA JWT signed with the RSA key, valid for 10 years
# Vault validates the RS256 algorithm, then delegates to mock-k8s for TokenReview
python3 - <<'EOF'
import jwt, time

payload = {
    "iss": "kubernetes/serviceaccount",
    "sub": "system:serviceaccount:default:builder-hub",
    "kubernetes.io/serviceaccount/namespace": "default",
    "kubernetes.io/serviceaccount/service-account.name": "builder-hub",
    "kubernetes.io/serviceaccount/service-account.uid": "builder-hub-dev-uid",
    "exp": int(time.time()) + 86400 * 365 * 10,
    "iat": int(time.time()),
}
key = open("/tmp/sa.key", "rb").read()
token = jwt.encode(payload, key, algorithm="RS256")
open("/vault-jwt/token", "w").write(token)
print("SA JWT written to /vault-jwt/token")
EOF

# Enable Kubernetes auth method
vault auth enable kubernetes

# Configure Kubernetes auth to use mock-k8s for TokenReview (no real k8s API needed)
vault write auth/kubernetes/config \
    kubernetes_host="http://mock-k8s:8443" \
    disable_iss_validation=true \
    disable_local_ca_jwt=true

# Create a policy granting read/write access to the KV secrets mount
vault policy write builder-hub - <<'POLICY'
path "secret/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
POLICY

# Create a role for the builder-hub service account
vault write auth/kubernetes/role/builder-hub \
    bound_service_account_names="builder-hub" \
    bound_service_account_namespaces="default" \
    token_policies="default,builder-hub" \
    token_ttl="24h" \
    token_max_ttl="48h"

echo "Vault Kubernetes auth configured."
