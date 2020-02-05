module github.com/devjoes/secret-sealer/plugin/devjoes/v1/secretsealer/

go 1.12

require (
	github.com/bitnami-labs/flagenv v0.0.0-20190607135054-a87af7a1d6fc
	github.com/bitnami-labs/pflagenv v0.0.0-20190702160147-b4d9f048d98f
	github.com/devjoes/sealed-secrets v0.9.8
	github.com/mattn/go-isatty v0.0.10
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	sigs.k8s.io/kustomize/api v0.3.2
	sigs.k8s.io/yaml v1.1.0
)

replace github.com/Azure/go-autorest/autorest/mocks => github.com/Azure/go-autorest/autorest/mocks v0.2.0
