# Brigade Docker Hub Gateway

![build](https://badgr.brigade2.io/v1/github/checks/brigadecore/brigade-dockerhub-gateway/badge.svg?appID=99005)
[![codecov](https://codecov.io/gh/brigadecore/brigade-dockerhub-gateway/branch/main/graph/badge.svg?token=91B1J1VKQH)](https://codecov.io/gh/brigadecore/brigade-dockerhub-gateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/brigadecore/brigade-dockerhub-gateway)](https://goreportcard.com/report/github.com/brigadecore/brigade-dockerhub-gateway)
[![slack](https://img.shields.io/badge/slack-brigade-brightgreen.svg?logo=slack)](https://kubernetes.slack.com/messages/C87MF1RFD)

<img width="100" align="left" src="logo.png">

This Brigade Docker Hub Gateway receives events
([webhooks](https://docs.docker.com/docker-hub/webhooks/)) from Docker Hub and
emits them into Brigade's event bus.

<br clear="left"/>

## Creating Webhooks

After [installation](docs/INSTALLATION.md), browse to any of your Docker Hub
repositories for which you'd like to send webhooks to this gateway. From the
tabs across the top of the page, select __Webhooks__. 

* In the __Webhook name__ field, add a meaningful name for the webhook.

* In the __Webhook URL__ field, use a value of the form
  `https://<DNS hostname or publicIP>/events?access_token=<url-encoded token>`.

* Click __Create__

> ⚠️&nbsp;&nbsp;Docker Hub doesn't provide _any_ reasonable mechanism for
> authenticating to the endpoints to which events (webhooks) are sent. Due to
> this, the only viable approach to authentication is to include a token (a
> shared secret) in the webhook URL as depicted above. Users are cautioned that
> even with TLS, this is not _entirely_ secure because web servers, reverse
> proxies, and other infrastructure are apt to capture entire URLs, including
> query parameters, in their access logs. _If your threat model suggests this is
> an intolerable degree of risk, then do not use this gateway and, more
> generally, do not use Docker Hub webhooks._

## Subscribing

Now subscribe any number of Brigade
[projects](https://docs.brigade.sh/topics/project-developers/projects/) to
events emitted by this gateway -- all of which have a value of
`brigade.sh/dockerhub` in their `source` field. You can subscribe to all event
types emitted by the gateway, or just specific ones.

In the example project definition below, we subscribe to `push` events, provided
they've originated from the fictitious `example-org/example` repository (see the
`repo` 
[qualifier](https://docs.brigade.sh/topics/project-developers/events/#qualifiers)).
You should adjust this value to match a repository for which you are sending
webhooks to your new gateway (see
[installation instructions](docs/INSTALLATION.md)).

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
      repo: example-org/example
  workerTemplate:
    defaultConfigFiles:
      brigade.js: |-
        const { events } = require("@brigadecore/brigadier");

        events.on("brigade.sh/dockerhub", "push", () => {
          console.log("Someone pushed an image to the example-org/example repository!");
        });

        events.process();
```

Assuming this file were named `project.yaml`, you can create the project like
so:

```shell
$ brig project create --file project.yaml
```

Pushing an image to the corresponding repo should now send a webhook from Docker
Hub to your gateway. The gateway, in turn, will emit the event into Brigade's
event bus. Brigade should initialize a worker (containerized event handler) for
every project that has subscribed to the event, and the worker should execute
the `brigade.js` script that was embedded in the example project definition.

List the events for the `dockerhub-demo` project to confirm this:

```shell
$ brig event list --project dockerhub-demo
```

Full coverage of `brig` commands is beyond the scope of this documentation, but
at this point,
[additional `brig` commands](https://docs.brigade.sh/topics/project-developers/brig/)
can be applied to monitor the event's status and view logs produced in the
course of handling the event.

## Events Received and Emitted by this Gateway

Docker Hub only supports one type of event (webhook) and that is the `push`
event.

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
