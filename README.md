# eks-lab-hello-api

Ultra basic Go HTTP API used across the EKS labs.

| Path | Purpose |
|---|---|
| `/` | `hello from <pod> (version <sha>)` |
| `/healthz` | liveness probe |
| `/readyz` | readiness probe; returns 503 after SIGTERM so the pod drains before exiting |
| `/burn?ms=200` | burns CPU for N ms (HPA lab) |

`DRAIN_DELAY` (default `10s`) sets how long the pod keeps serving after SIGTERM.

## CI

When you push to `main`, `.github/workflows/build.yml`:

1. assumes an AWS role through GitHub OIDC (no AWS keys stored in GitHub),
2. builds `linux/amd64` + `linux/arm64` and pushes to ECR with the short SHA as the tag,
3. updates `newTag` in `eks-lab-gitops/workloads/hello-api/kustomization.yaml`, and Argo CD then rolls the new version out.

## Run locally

```bash
go run .
curl localhost:8080/
```

The repo was seeded by Terraform (`eks-interview-prep/labs/00-bootstrap`). After that it's a normal repo.
