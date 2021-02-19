# k8s development Readme

The application ships with a basic k8s config in the `k8s/` directory.  There, you will find an `api` helm chart and a `values.yaml` to deploy the pod, service and ingress.  By default, `skaffold` will use the [paketo buildpacks](https://paketo.io/) and will reference the configuration in `config/config.json`.

## install docker desktop and enable kubernetes

* [Install docker desktop](https://www.docker.com/products/docker-desktop)

* [Enable kubernetes on docker desktop](https://docs.docker.com/docker-for-mac/#kubernetes)

## install skaffold

[Install skaffold](https://skaffold.dev/docs/getting-started/#installing-skaffold)

## setup ingress controller (do this once on your cluster)

https://kubernetes.github.io/ingress-nginx/deploy/

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v0.41.2/deploy/static/provider/cloud/deploy.yaml
```

## develop

* run `skaffold dev` in the root of the project

* update your `hosts` file to point spindev.internal.yale.edu to localhost

* use the endpoint `http://<<spindev.internal.yale.edu>>/v1/<apiname>`

Saving your code should rebuild and redeploy your project automatically

## [non-]profit
