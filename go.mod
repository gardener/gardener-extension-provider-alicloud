module github.com/gardener/gardener-extension-provider-alicloud

go 1.15

require (
	github.com/Masterminds/semver v1.5.0
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.442
	github.com/aliyun/aliyun-oss-go-sdk v2.0.1+incompatible
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/coreos/go-systemd/v22 v22.1.0
	github.com/frankban/quicktest v1.9.0 // indirect
	github.com/gardener/etcd-druid v0.3.0
	github.com/gardener/gardener v1.16.0
	github.com/gardener/machine-controller-manager v0.36.0
	github.com/go-logr/logr v0.3.0
	github.com/gobuffalo/packr/v2 v2.8.1
	github.com/golang/mock v1.4.4-0.20200731163441-8734ec565a4d
	github.com/golang/snappy v0.0.2 // indirect
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/pierrec/lz4 v2.5.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.8.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/ulikunitz/xz v0.5.7 // indirect
	k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery v0.19.6
	k8s.io/apiserver v0.19.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.19.6
	k8s.io/component-base v0.19.6
	k8s.io/gengo v0.0.0-20200428234225-8167cfdcfc14
	k8s.io/helm v2.16.1+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.19.6
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/controller-runtime v0.7.1
)

replace (
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.19.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.6
	k8s.io/apiserver => k8s.io/apiserver v0.19.6
	k8s.io/client-go => k8s.io/client-go v0.19.6
	k8s.io/code-generator => k8s.io/code-generator v0.19.6
	k8s.io/component-base => k8s.io/component-base v0.19.6
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
)
