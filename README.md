# [Gardener Extension for Alicloud provider](https://gardener.cloud)

[![REUSE status](https://api.reuse.software/badge/github.com/gardener/gardener-extension-provider-alicloud)](https://api.reuse.software/info/github.com/gardener/gardener-extension-provider-alicloud)
[![CI Build status](https://concourse.ci.gardener.cloud/api/v1/teams/gardener/pipelines/gardener-extension-provider-alicloud-master/jobs/master-head-update-job/badge)](https://concourse.ci.gardener.cloud/teams/gardener/pipelines/gardener-extension-provider-alicloud-master/jobs/master-head-update-job)
[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardener-extension-provider-alicloud)](https://goreportcard.com/report/github.com/gardener/gardener-extension-provider-alicloud)

Project Gardener implements the automated management and operation of [Kubernetes](https://kubernetes.io/) clusters as a service.
Its main principle is to leverage Kubernetes concepts for all of its tasks.

Recently, most of the vendor specific logic has been developed [in-tree](https://github.com/gardener/gardener).
However, the project has grown to a size where it is very hard to extend, maintain, and test.
With [GEP-1](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md) we have proposed how the architecture can be changed in a way to support external controllers that contain their very own vendor specifics.
This way, we can keep Gardener core clean and independent.

This controller implements Gardener's extension contract for the Alicloud provider.

An example for a `ControllerRegistration` resource that can be used to register this controller to Gardener can be found [here](example/controller-registration.yaml).

Please find more information regarding the extensibility concepts and a detailed proposal [here](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md).

## Supported Kubernetes versions

This extension controller supports the following Kubernetes versions:

| Version         | Support | Conformance test results |
|-----------------|---------| ------------------------ |
| Kubernetes 1.33 | 1.33.0+ | N/A |
| Kubernetes 1.32 | 1.32.0+ | [![Gardener v1.32 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.32%20Alibaba%20Cloud/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.32%20Alibaba%20Cloud) |
| Kubernetes 1.31 | 1.31.0+ | [![Gardener v1.31 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.31%20Alibaba%20Cloud/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.31%20Alibaba%20Cloud) |
| Kubernetes 1.30 | 1.30.0+ | [![Gardener v1.30 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.30%20Alibaba%20Cloud/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.30%20Alibaba%20Cloud) |
| Kubernetes 1.29 | 1.29.0+ | [![Gardener v1.29 Conformance Tests](https://testgrid.k8s.io/q/summary/conformance-gardener/Gardener,%20v1.29%20Alibaba%20Cloud/tests_status?style=svg)](https://testgrid.k8s.io/conformance-gardener#Gardener,%20v1.29%20Alibaba%20Cloud) |

Please take a look [here](https://github.com/gardener/gardener/blob/master/docs/usage/shoot-operations/supported_k8s_versions.md) to see which versions are supported by Gardener in general.

----

## How to start using or developing this extension controller locally

You can run the controller locally on your machine by executing `make start`.

Static code checks and tests can be executed by running `make verify`. We are using Go modules for Golang package dependency management and [Ginkgo](https://github.com/onsi/ginkgo)/[Gomega](https://github.com/onsi/gomega) for testing.

## Feedback and Support

Feedback and contributions are always welcome!

Please report bugs or suggestions as [GitHub issues](https://github.com/gardener/gardener-extension-provider-alicloud/issues) or reach out on [Slack](https://gardener-cloud.slack.com/) (join the workspace [here](https://gardener.cloud/community/community-bio/)).

## Learn more!

Please find further resources about out project here:

* [Our landing page gardener.cloud](https://gardener.cloud/)
* ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
* ["Gardener Project Update" blog on kubernetes.io](https://kubernetes.io/blog/2019/12/02/gardener-project-update/)
* [GEP-1 (Gardener Enhancement Proposal) on extensibility](https://github.com/gardener/gardener/blob/master/docs/proposals/01-extensibility.md)
* [GEP-4 (New `core.gardener.cloud/v1beta1` API)](https://github.com/gardener/gardener/blob/master/docs/proposals/04-new-core-gardener-cloud-apis.md)
* [Extensibility API documentation](https://github.com/gardener/gardener/tree/master/docs/extensions)
* [Gardener Extensions Golang library](https://godoc.org/github.com/gardener/gardener/extensions/pkg)
* [Gardener API Reference](https://gardener.cloud/api-reference/)

