module github.com/devjoes/secret-sealer

go 1.12

require (
	github.com/devjoes/sealed-secrets v0.9.9
	github.com/pkg/errors v0.8.1
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.2.0

