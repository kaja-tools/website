# kaja.tools

This repo is the source for the [kaja.tools](https://kaja.tools) website.

## Commands

```
kubectl cp apps/users/proto/users.proto kaja-deployment-7c99d757c4-6sjgq:/workspace/users.proto

kubectl kustomize k8s/overlays/production
kubectl apply -k k8s/overlays/production
kubectl rollout restart deployment kaja-deployment
kubectl apply -k k8s/overlays/development
```
