module github.com/sacloud/sakura-cloud-controller-manager

go 1.13

require (
	github.com/ghodss/yaml v0.0.0-20180820084758-c7ce16629ff4
	github.com/hashicorp/go-multierror v1.0.0
	github.com/imdario/mergo v0.3.5
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sacloud/libsacloud v1.27.1
	github.com/stretchr/testify v1.2.2
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/apiserver v0.0.0
	k8s.io/cloud-provider v0.0.0
	k8s.io/component-base v0.0.0
	k8s.io/klog v0.3.1
	k8s.io/kubernetes v1.15.5
	k8s.io/utils v0.0.0-20190221042446-c2654d5206da
)

replace k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d

replace k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190918200908-1e17798da8c1

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190918202139-0b14c719ca62

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190918203125-ae665f80358a

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190918202959-c340507a5d48

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190612205613-18da4a14b22b

replace k8s.io/component-base => k8s.io/component-base v0.0.0-20190918200425-ed2f0867c778

replace k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190817025403-3ae76f584e79

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20190918203248-97c07dcbb623

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190918201136-c3a845f1fbb2

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20190918202837-c54ce30c680e

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20190918202429-08c8357f8e2d

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20190918202713-c34a54b3ec8e

replace k8s.io/kubelet => k8s.io/kubelet v0.0.0-20190918202550-958285cf3eef

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20190918203421-225f0541b3ea

replace k8s.io/metrics => k8s.io/metrics v0.0.0-20190918202012-3c1ca76f5bda

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20190918201353-5cc279503896
