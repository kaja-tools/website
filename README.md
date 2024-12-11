# website
https://medium.com/@jainchirag8001/tls-on-aks-ingress-with-letsencrypt-f42d65725a3

```
kubectl cp apps/users/proto/users.proto kaja-deployment-7c99d757c4-6sjgq:/workspace/users.proto

kubectl kustomize k8s/overlays/production
kubectl apply -k k8s/overlays/production
kubectl apply -k k8s/overlays/development
```