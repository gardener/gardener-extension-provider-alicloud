module github.com/gardener/gardener-extension-provider-alicloud

go 1.14

require (
	github.com/Masterminds/semver v1.5.0
	github.com/ahmetb/gen-crd-api-reference-docs v0.1.5
	github.com/aliyun/alibaba-cloud-sdk-go v1.60.340
	github.com/aliyun/aliyun-oss-go-sdk v2.0.1+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/frankban/quicktest v1.9.0 // indirect
	github.com/gardener/etcd-druid v0.3.0
	github.com/gardener/gardener v1.7.0
	github.com/gardener/machine-controller-manager v0.27.0
	github.com/go-logr/logr v0.1.0
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/golang/mock v1.4.3
	github.com/kubernetes-csi/external-snapshotter v1.2.2
	github.com/nwaples/rardecode v1.1.0 // indirect
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pierrec/lz4 v2.5.1+incompatible // indirect
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/ulikunitz/xz v0.5.7 // indirect
	golang.org/x/sys v0.0.0-20200420163511-1957bb5e6d1f // indirect
	k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery v0.17.6
	k8s.io/apiserver v0.17.6
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.17.6
	k8s.io/component-base v0.17.6
	k8s.io/gengo v0.0.0-20190826232639-a874a240740c
	k8s.io/helm v2.16.1+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.17.6
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8
	sigs.k8s.io/controller-runtime v0.5.5
)

replace (
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.17.6
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.6
	k8s.io/apiserver => k8s.io/apiserver v0.17.6
	k8s.io/client-go => k8s.io/client-go v0.17.6
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.6
	k8s.io/code-generator => k8s.io/code-generator v0.17.6
	k8s.io/component-base => k8s.io/component-base v0.17.6
	k8s.io/helm => k8s.io/helm v2.13.1+incompatible
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.6
)
