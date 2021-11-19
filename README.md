# Brigade Docker Hub Gateway

![build](https://badgr.brigade2.io/v1/github/checks/brigadecore/brigade-dockerhub-gateway/badge.svg?appID=99005)
[![codecov](https://codecov.io/gh/brigadecore/brigade-dockerhub-gateway/branch/main/graph/badge.svg?token=91B1J1VKQH)](https://codecov.io/gh/brigadecore/brigade-dockerhub-gateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/brigadecore/brigade-dockerhub-gateway)](https://goreportcard.com/report/github.com/brigadecore/brigade-dockerhub-gateway)
[![slack](https://img.shields.io/badge/slack-brigade-brightgreen.svg?logo=slack)](https://kubernetes.slack.com/messages/C87MF1RFD)

<img width="100" align="left" src="logo.png">

This is a work-in-progress
[Brigade 2](https://github.com/brigadecore/brigade/tree/v2)
compatible gateway that receives events (webhooks) from Docker Hub
and propagates them into Brigade 2's event bus.

<br clear="left"/>

## Installation

Prerequisites:

* A Kubernetes cluster:
    * For which you have the `admin` cluster role
    * That is already running Brigade 2
    * Capable of provisioning a _public IP address_ for a service of type
      `LoadBalancer`. (This means you won't have much luck running the gateway
      locally in the likes of kind or minikube unless you're able and willing to
      mess with port forwarding settings on your router, which we won't be
      covering here.)

* `kubectl`, `helm` (commands below require Helm 3.7.0+), and `brig` (the
  Brigade 2 CLI)

### 1. Create a Service Account for the Gateway

__Note:__ To proceed beyond this point, you'll need to be logged into Brigade 2
as the "root" user (not recommended) or (preferably) as a user with the `ADMIN`
role. Further discussion of this is beyond the scope of this documentation.
Please refer to Brigade's own documentation.

Using Brigade 2's `brig` CLI, create a service account for the gateway to use:

```console
$ brig service-account create \
    --id brigade-dockerhub-gateway \
    --description brigade-dockerhub-gateway
```

Make note of the __token__ returned. This value will be used in another step.
_It is your only opportunity to access this value, as Brigade does not save it._

Authorize this service account to create new events:

```console
$ brig role grant EVENT_CREATOR \
    --service-account brigade-dockerhub-gateway \
    --source brigade.sh/dockerhub
```

__Note:__ The `--source brigade.sh/dockerhub` option specifies that
this service account can be used _only_ to create events having a value of
`brigade.sh/dockerhub` in the event's `source` field. _This is a
security measure that prevents the gateway from using this token for
impersonating other gateways._

### 2. Install the Docker Hub Gateway

For now, we're using the [GitHub Container Registry](https://ghcr.io) (which is
an [OCI registry](https://helm.sh/docs/topics/registries/)) to host our Helm
chart. Helm 3.7 has _experimental_ support for OCI registries. In the event that
the Helm 3.7 dependency proves troublesome for users, or in the event that this
experimental feature goes away, or isn't working like we'd hope, we will revisit
this choice before going GA.

First, be sure you are using
[Helm 3.7.0](https://github.com/helm/helm/releases/tag/v3.7.0) or greater and
enable experimental OCI support:

```console
$ export HELM_EXPERIMENTAL_OCI=1
```

As this chart requires custom configuration as described above to function
properly, we'll need to create a chart values file with said config.

Use the following command to extract the full set of configuration options into
a file you can modify:

```console
$ helm inspect values oci://ghcr.io/brigadecore/brigade-dockerhub-gateway \
    --version v0.3.0 > ~/brigade-dockerhub-gateway-values.yaml
```

Edit `~/brigade-dockerhub-gateway-values.yaml`, making the following changes:

* `host`: Set this to the host name where you'd like the gateway to be
  accessible.

* `brigade.apiAddress`: Address of the Brigade API server, beginning with
  `https://`

* `brigade.apiToken`: Service account token from step 2

* `service.type`: If you plan to enable ingress (advanced), you can leave this
  as its default -- `ClusterIP`. If you do not plan to enable ingress, you
  probably will want to change this value to `LoadBalancer`.

* `tokens`: This field should define tokens that can be used by clients to send
  events (webhooks) to this gateway. Note that keys are completely ignored by
  the gateway and only the values (tokens) matter. The keys only serve as
  recognizable token identifiers for human operators.

Save your changes to `~/brigade-dockerhub-gateway-values.yaml` and use the
following command to install the gateway using the above customizations:

```console
$ helm install brigade-dockerhub-gateway \
    oci://ghcr.io/brigadecore/brigade-dockerhub-gateway
    --version v0.3.0 \
    --create-namespace \
    --namespace brigade-dockerhub-gateway \
    --values ~/brigade-dockerhub-gateway-values.yaml \
    --wait \
    --timeout 300s
```

### 3. (RECOMMENDED) Create a DNS Entry

If you overrode defaults and set `service.type` to `LoadBalancer`, use this
command to find the gateway's public IP address:

```console
$ kubectl get svc brigade-dockerhub-gateway \
    --namespace brigade-dockerhub-gateway \
    --output jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

If you overrode defaults and enabled support for an ingress controller, you
probably know what you're doing well enough to track down the correct IP without
our help. 😉

With this public IP in hand, edit your name servers and add an `A` record
pointing your domain to the public IP.

### 4. Create Webhooks

In your browser, go to your Docker Hub repository for which you'd like to send
webhooks to this gateway. From the tabs across the top of the page, select
__Webhooks__. 

* In the __Webhook name__ field, add a meaningful name for the webhook.

* In the __Webhook URL__ field, use a value of the form `https://<DNS hostname or publicIP>/events?access_token=<url-encoded token>`.

* Click __Create__

__Note:__ Docker Hub doesn't provide _any_ reasonable mechanism for
authenticating to the endpoints to which events (webhooks) are sent. Due to
this, the only viable approach to authentication is to include a token (a shared
secret) in the webhook URL as depicted above. Users are cautioned that even with
TLS in play, this is not _entirely_ secure because web servers, reverse proxies,
and other infrastructure are apt to capture entire URLs, including query
parameters, in their access logs. _If your threat model suggests this is an
intolerable degree of risk, then do not use this gateway and, more generally, do
not use Docker Hub webhooks._

### 5. Add a Brigade Project

You can create any number of Brigade projects (or modify an existing one) to
listen for events that were sent from a Docker Hub repository to your gateway
and, in turn, emitted into Brigade's event bus. You can subscribe to all event
types emitted by the gateway, or just specific ones.

In the example project definition below, we subscribe to all events emitted by
the gateway, provided they've originated from the fictitious
`example-org/example-repo` repository (see the `repo` qualifier).

```yaml
apiVersion: brigade.sh/v2
kind: Project
metadata:
  id: dockerhub-demo
description: A project that demonstrates integration with Docker Hub
spec:
  eventSubscriptions:
  - source: brigade.sh/dockerhub
    types:
    - *
    qualifiers:
      repo: example-org/example-repo
  workerTemplate:
    defaultConfigFiles:
      brigade.js: |-
        const { events } = require("@brigadecore/brigadier");

        events.on("brigade.sh/dockerhub", "push", () => {
          console.log("Someone pushed an image to the example-org/example-repo repository!");
        });

        events.process();
```

In the alternative example below, we subscribe _only_ to `push` events:

```yaml
apiVersion: brigade.sh/v2
kind: Project
metadata:
  id: dockerhub-demo
description: A project that demonstrates integration with Docker Hub
spec:
  eventSubscriptions:
  - source: brigade.sh/dockerhub
    types:
    - push
    qualifiers:
      repo: example-org/example-repo
  workerTemplate:
    defaultConfigFiles:
      brigade.js: |-
        const { events } = require("@brigadecore/brigadier");

        events.on("brigade.sh/dockerhub", "push", () => {
          console.log("Someone pushed an image to the example-org/example-repo repository!");
        });

        events.process();
```

Assuming this file were named `project.yaml`, you can create the project like
so:

```console
$ brig project create --file project.yaml
```

Push an image to the Docker Hub repo for which you configured webhooks to send
an event (webhook) to your gateway. The gateway, in turn, will emit the event
into Brigade's event bus. Brigade should initialize a worker (containerized
event handler) for every project that has subscribed to the event, and the
worker should execute the `brigade.js` script that was embedded in the example
project definition.

List the events for the `dockerhub-demo` project to confirm this:

```console
$ brig event list --project dockerhub-demo
```

Full coverage of `brig` commands is beyond the scope of this documentation, but
at this point, additional `brig` commands can be applied to monitor the event's
status and view logs produced in the course of handling the event.

## Events Received and Emitted by this Gateway

Events received by this gateway from Docker Hub are, in turn, emitted into
Brigade's event bus.

Docker Hub only supports one type of event (webhook) and that is the `push`
event.

## Examples Projects

See `examples/` for complete Brigade projects that demonstrate various
scenarios.

## Contributing

The Brigade project accepts contributions via GitHub pull requests. The
[Contributing](CONTRIBUTING.md) document outlines the process to help get your
contribution accepted.

## Support & Feedback

We have a slack channel!
[Kubernetes/#brigade](https://kubernetes.slack.com/messages/C87MF1RFD) Feel free
to join for any support questions or feedback, we are happy to help. To report
an issue or to request a feature open an issue
[here](https://github.com/brigadecore/brigade-dockerhub-gateway/issues)

## Code of Conduct

Participation in the Brigade project is governed by the
[CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).
