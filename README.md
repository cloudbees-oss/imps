# IMPS

**WARNING**: CloudBees maintains this fork of the original repository and applies only essential security fixes to support its customers. The use of the images or Helm chart published in this project is **NOT** supported for external purposes. Utilizing these images or charts is at your own risk.

IMPS (IMagePullSecrets) controller is a Kubernetes operator that manages pull secrets based label or annotation
selectors. It natively supports ECR and standard docker configuration secrets.

IMPS provides two modes of operation:
- IMPS controller: a full fledged solution for managing and refreshing secrets in multiple namespaces
- IMPS token refresher: a small utility that allows to refresh ECR tokens inside one namespace

## Using the token refresher

The token refresher Docker images are available in the `ghcr.io/banzaicloud/imagepullsecrets-refresher` Docker Registry.

The refresher uses the same kind of secrets as we will discuss in regard the Controller, however it can only output pull secrets into a single namespace.

For example a Pod running the refresher with the following arguments:

```shell
/manager
    --target-secret=default.ecr-image-pull-secrets
    --source-secret=default.ecr-credentials-1
    --source-secret=default.ecr-credentials-2
```

will ensure that the AWS credentials stored inside the `default` namespaces `ecr-credentails-1` and `ecr-credentials-2` secrets are used to fetch
ECR tokens for the set registries, and the image pull secret will be put inside the `ecr-image-pull-secrets` secret inside the `default` namespace.

The format of AWS credentials needed by the refresher is the following:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ecr-credentials-1
  namespace: default
type: banzaicloud.io/aws-ecr-login-config
stringData:
  accessKeyID: XXX # AWS AccessKeyID
  secretKey: XXXX # AWS SecretAccessKey
  region: us-east-1 # ECR repository's region to use the token for
  accountID: "123456789"  # ECR repository's account ID to use the token for
  roleArn: "arn:aws:iam::0000000000000:role/ECR-PulllAccess" # Optional, use to assume role in cross account environments
```

The secret type should be `banzaicloud.io/aws-ecr-login-config`.

*Note*: the refresher needs list and watch Cluster permissions for secrets, and read access to the source secrets, and created/delete/update for the target secret.

If there's interest we can provide a helm chart for the refresher too, please create an issue if you are interested.

## Motivation for the Controller

The solution is geared towards Istio or any solution relying on sidecar injection. The issue is
that the sidecar injection can happen in any namespace the user starts a `Deployment` or `StatefulSet`
(`Workload` from now on). Assuming that the image is inside a private docker registry the `Pod` of `Workload` needs to
have access to that registry.

ImagePullSecrets can be specified at many places, like inside a `ServiceAccount`, however the sidecar injector should not
change the `Pod`'s service account. What it can do, is to add a new `Secret` reference inside the `Pod`'s
`imagePullSecrets` list. Unfortunately you cannot specify a namespace field in that list, meaning that the `Secret`
needs to reside in the same `Namespace` as the `Pod`.

As we did not want to implement this feature inside our sidecar injector, we came up with the concept of IMPS.
For example the following `CustomResource` instructs IMPS to create a secret called `istio-pull-secret` inside each
`Namespace` with the annotation of `sidecar.istio.io/inject=true` or if any Pod inside a namespace has the same
annotation:

```yaml
apiVersion: images.banzaicloud.io/v1alpha1
kind: ImagePullSecret
metadata:
  name: imps
spec:
  registry:
    credentials:
      - name: registry--https-index.docker.io-v1-pull-secret-5cbe024b
        namespace: registry-access
  target:
    namespaces:
      annotations:
        - matchAnnotations:
            sidecar.istio.io/inject: "true"
    namespacesWithPods:
      annotations:
        - matchAnnotations:
            sidecar.istio.io/inject: "true"
    secret:
      name: istio-pull-secret
```

The secret will contain the docker login config from the `registry--https-index.docker.io-v1-pull-secret-5cbe024b`
secret from the `registry-access` namespace.

Given that this CR is controlling the IMPS controller any time a new `Pod` starts, or a `Namespace` gets created the
secret will be injected automatically.

## Installation

Helm chart's source is available in `deploy/imagepullsecrets`.

The helm chart can also be downloaded from the `banzaicloud-stable` helm repository:

```shell
helm repo add banzaicloud-stable https://kubernetes-charts.banzaicloud.com
helm install imps banzaicloud-stable/imagepullsecrets
```

## Features

### Docker login credentials support

The secrets specified in the `.spec.registry.credentials` array can be pointed to a secret that is created the same way
as described in [Pull an Image from a Private Registry](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).

If multiple credentials are specified, their contents will be merged into a single `kubernetes.io/dockerconfigjson` typed
`Secret`.

### ECR Support

IMPS supports logging in to [ECR](https://aws.amazon.com/ecr/) registries. This can be useful if you need to access an
ECR registry from outside of AWS, as ECR issues tokens that are only valid for 12 hours, then they need to be renewed
based on IAM credentials.

To add an ECR repository login credential create a secret based on this template:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ecr-pull-secret
  namespace: registry-access
type: banzaicloud.io/aws-ecr-login-config
stringData:
  accessKeyID: XXX # AWS AccessKeyID
  secretKey: XXXX # AWS SecretAccessKey
  region: us-east-1 # ECR repository's region to use the token for
  accountID: "123456789"  # ECR repository's account ID to use the token for
  roleArn: "arn:aws:iam::0000000000000:role/ECR-PulllAccess" # Optional, use to assume role in cross account environments
```

*Note*: Make sure `accountID` is given as string and not bare numbers, Kubernetes Secret's `stringData` field will only accept strings.

### Using IMPS to provision secrets in selected namespaces

The following CR will provision the `.spec.registry.credentials` login credentials in the namespaces listed under the
`.spec.target.names.`:

```yaml
apiVersion: images.banzaicloud.io/v1alpha1
kind: ImagePullSecret
metadata:
  name: imps
spec:
  registry:
    credentials:
      - name: registry--https-index.docker.io-v1-pull-secret-5cbe024b
        namespace: registry-access
      - name: registry--999653XXXXX.dkr.ecr.eu-central-1.amazonaws.com-pull-secret-5e3481fc
        namespace: registry-access
  target:
    namespaces:
      names:
        - backyards-registry-access
        - backyards-system
        - backyards-canary
        - cert-manager
        - istio-system
    secret:
      name: istio-pull-secret
```

This example CR instructs the controller to:
- load the `registry--https-index.docker.io-v1-pull-secret-5cbe024b` secret from the `registry-access` namespace
- load the `registry--999653XXXXX.dkr.ecr.eu-central-1.amazonaws.com-pull-secret-5e3481fc` ECR credential-based secret from the `registry-access` namespace
  - Given that this is an ECR typed secret, it will call the AWS API to obtain a login token an generate the resulting docker authentication config
- merges all of the docker authentication resulting from the previous steps
- writes the resulting configuration into the `istio-pull-secret` in the following namespaces:
  - `backyards-registry-access`
  - `backyards-system`
  - `backyards-canary`
  - `cert-manager`
  - `istio-system`

If the ECR credentials are to expire the controller will automatically execute the previous steps again to ensure that the tokens are still valid.

### Rule evaulation logic

As it is visible from the previous examples both the `.spec.namespaces.annotations` and `.spec.namespaces.labels` are
arrays. The intent behind is that a namespace is selected for inclusion of the pull secret if *ANY* of those conditions
match (logical OR).

For example:
```yaml
apiVersion: images.banzaicloud.io/v1alpha1
kind: ImagePullSecret
metadata:
  name: imps
spec:
  # ...
  target:
    namespaces:
      annotations:
        - matchAnnotations:
            sidecar.istio.io/inject: "true"
        - matchAnnotations:
            backyards.banzaicloud.io/image-registry-access: "true"
      labels:
        - matchExpressions:
            - key: istio.io/rev
              operator: Exists
      names:
        - backyards-registry-access
        - backyards-system
        - backyards-canary
        - cert-manager
        - istio-system
    namespacesWithPods:
      annotations:
        - matchAnnotations:
            sidecar.istio.io/inject: "true"
      labels:
        - matchExpressions:
            - key: istio.io/rev
              operator: Exists
    secret:
      name: istio-pull-secret
```

This CR instructs IMPS to provision `istio-pull-secret` if **any** of the following conditions is true:
- The namespace has an annotation of `sidecar.istio.io/inject=true` OR
- The namespace has an annotation of `backyards.banzaicloud.io/image-registry-access=true` OR
- The namespace has a label with key `istio.io/rev` OR
- The namespace's name is listed in the names part OR
- A pod exists inside the namespace that has an annotation of `sidecar.istio.io/inject=true` OR
- A pod exists inside the namespace that has a label with key `istio.io/rev`

## Troubleshooting

In case there is an issue with the controller the imps CR shows the reason of the issue. For example:

```bash
# kubectl get imps
NAME                                       STATE    RECONCILED   VALIDITY SECONDS   SECRET NAME           NAMESPACES
imps-imagepullsecrets-controller-default   Failed   21m          43161              default-secret-name   ["default"]
```

The `Failed` string indicates that something is wrong when creating the target secrets. To check the details, please describe the given IMPS CR:
```bash
# kubectl describe imps imps-imagepullsecrets-controller-default
Name:         imps-imagepullsecrets-controller-default
Namespace:
...
Status:
  Last Successful Reconciliation:  2021-10-25T11:39:18Z
  Managed Namespaces:
    default
  Reason:  some source secrets failed to render
  Source Secret Status:
    default.test:    Ok
    default.test2:   operation error ECR: GetAuthorizationToken, https response error StatusCode: 400, RequestID: fa6b9762-522d-4aee-bb6d-69280ed22ae7, api error UnrecognizedClientException: The security token included in the request is invalid.
  Status:            Failed
  Validity Seconds:  43161
Events:
  Type     Reason                  Age                From                         Message
  ----     ------                  ----               ----                         -------
  Warning  SourceCredentialsError  30m (x2 over 30m)  imagepullsecrets-controller  some source secrets failed to render
  Warning  SourceCredentialsError  27m (x2 over 27m)  imagepullsecrets-controller  Source cerdentials failed to process: some source secrets failed to render
  Warning  SourceCredentialsError  22m (x3 over 23m)  imagepullsecrets-controller  Source cerdentials failed to process: [default.test2]
  ```

The `Status.Status` field indicates that the reconciliation has failed. The `Status.Reason` shows the failure case (`some source secrets failed to render`). The `Status.Source Secret Status` indicates what failed during the reconciliation:
```
 Source Secret Status:
    default.test:    Ok
    default.test2:   operation error ECR: GetAuthorizationToken, https response error StatusCode: 400, RequestID: fa6b9762-522d-4aee-bb6d-69280ed22ae7, api error UnrecognizedClientException: The security token included in the request is invalid.
```

As it is visible here the `default` namespace's `test2` source credential has failed.

## Similar projects

Other projects focusing on this problem space:
- https://github.com/SUSE/registries-operator
- https://github.com/titansoft-pte-ltd/imagepullsecret-patcher

## License

The project is licensed under the [Apache 2.0 License](LICENSE).
