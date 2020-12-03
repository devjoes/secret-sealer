package main

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/devjoes/sealed-secrets/pkg/apis/sealed-secrets/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/cert"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	pluginHelper          *resmap.PluginHelpers
	Target                types.Selector `json:"target,omitempty" yaml:"target,omitempty"`
	Cert                  string         `json:"cert,omitempty" yaml:"cert,omitempty"`
	Verbose               bool           `json:"verbose,omitempty" yaml:"verbose,omitempty"`
	CopyLabelsAnnotations bool           `json:"copyLabelsAnnotations,omitempty" yaml:"copyLabelsAnnotations,omitempty"`
}

//KustomizePlugin not used - but apparently required
var KustomizePlugin plugin

func (p *plugin) Config(ph *resmap.PluginHelpers, c []byte) (err error) {
	p.debug("config start")
	p.Target = types.Selector{
		Name: "",
		Gvk: resid.Gvk{
			Kind: "Secret",
		},
	}
	p.Cert = ""
	p.pluginHelper = ph
	err = yaml.Unmarshal(c, p)

	if err != nil {
		p.debug("config error")
		return err
	}
	p.debug("config end")
	return err
}

func (p *plugin) checkOptions() error {
	if p.Cert == "" {
		return errors.New("Cert option is required")
	}
	return nil
}

func (p *plugin) Transform(rm resmap.ResMap) error {
	p.debug("transform start")
	err := p.checkOptions()
	if err != nil {
		return err
	}
	secrets, err := p.extractAndRemoveSecrets(rm, p.Target)
	if err != nil {
		p.debug("transform error")
		return err
	}

	for _, res := range secrets {
		if res.GetKind() == "Secret" {
			sSecret, err := p.sealSecret(&res)
			if err != nil {
				p.debug("transform error")
				return err
			}
			p.debug("Appending secret %v", sSecret)
			rm.Append(&sSecret)
		}
	}
	p.debug("transform end")
	return nil
}

func (p *plugin) sealSecret(secret *resource.Resource) (resource.Resource, error) {
	p.debug("sealSecret start %v", secret.GetGvk())
	k8sSecret, err := prepSecretForSealing(secret)
	if err != nil {
		return resource.Resource{}, err
		//return resource.Resource{}, errors.Wrap(err, "Error converting kustomize Secret in to native k8s Secret")
	}
	sealedYaml, err := p.callKubeSealAPI(&k8sSecret)

	if err != nil {
		return resource.Resource{}, errors.Wrap(err, "Error calling kubeseal")
	}
	sSecretArray, err := p.pluginHelper.ResmapFactory().NewResMapFromBytes(sealedYaml)
	if err != nil {
		return resource.Resource{}, err
	}
	if sSecretArray.Size() != 1 {
		return resource.Resource{}, errors.New(fmt.Sprintf("Expected a single SealedSecret but received %d", sSecretArray.Size()))
	}
	sSecret := sSecretArray.GetByIndex(0)
	if p.CopyLabelsAnnotations {
		sSecret.SetLabels(joinMaps(sSecret.GetLabels(), secret.GetLabels(), "type", "generated"))
		sSecret.SetAnnotations(joinMaps(sSecret.GetAnnotations(), secret.GetAnnotations(), "note", "generated"))
	}

	p.debug("sealSecret end")
	return *sSecret, nil
}

func joinMaps(a map[string]string, b map[string]string, excludeKey string, excludeValue string) map[string]string {
	result := make(map[string]string, len(a)+len(b))

	for k, v := range a {
		if k != excludeKey && v != excludeValue {
			result[k] = v
		}
	}
	for k, v := range b {
		if k != excludeKey && v != excludeValue {
			result[k] = v
		}
	}
	return result
}

func prepSecretForSealing(secret *resource.Resource) (v1.Secret, error) {
	v1Secret := v1.Secret{}
	if secret.GetNamespace() == "" {
		secret.SetNamespace("default")
	}
	secretJSON, err := secret.MarshalJSON()
	if err != nil {
		return v1Secret, err
	}

	reader := bytes.NewReader(secretJSON)
	if err := json.NewDecoder(reader).Decode(&v1Secret); err != nil {
		return v1Secret, err
	}
	// Strip read-only server-side ObjectMeta (if present)
	v1Secret.SetSelfLink("")
	v1Secret.SetUID("")
	v1Secret.SetResourceVersion("")
	v1Secret.Generation = 0
	v1Secret.SetCreationTimestamp(metav1.Time{})
	v1Secret.SetDeletionTimestamp(nil)
	v1Secret.DeletionGracePeriodSeconds = nil

	return v1Secret, err
}

func (p *plugin) callKubeSealAPI(secret *v1.Secret) ([]byte, error) {
	p.debug("callKubeSealAPI start")
	info, ok := runtime.SerializerInfoForMediaType(scheme.Codecs.SupportedMediaTypes(), runtime.ContentTypeYAML)
	encoder := scheme.Codecs.EncoderForVersion(info.Serializer, v1alpha1.SchemeGroupVersion)
	if !ok {
		return nil, errors.New("SerializerInfoForMediaType Failed")
	}

	p.debug("getting cert")
	cert, err := openCertLocal(p.Cert)
	defer cert.Close()
	p.debug("got cert")
	if err != nil {
		return nil, errors.Wrapf(err, "Error opening cert %s", p.Cert)
	}

	key, err := parseKey(cert)
	if err != nil {
		return nil, errors.Wrapf(err, "Error extracting key from cert %s", p.Cert)
	}

	sessionKeySeed := os.Getenv("SESSION_KEY_SEED")

	sealedSecret, err := v1alpha1.NewSealedSecret(scheme.Codecs, key, secret, sessionKeySeed)
	if err != nil {
		return nil, errors.Wrapf(err, "Error sealing secret %s in %s", secret.Name, secret.Namespace)
	}
	buf, err := runtime.Encode(encoder, sealedSecret)
	if err != nil {
		return nil, err
	}

	p.debug("callKubeSealAPI end")
	return buf, nil
}

func (p *plugin) extractAndRemoveSecrets(rm resmap.ResMap, selector types.Selector) ([]resource.Resource, error) {
	p.debug("extractAndRemoveSecrets start")
	found, err := rm.Select(selector)
	if err != nil {
		return nil, err
	}

	var result = []resource.Resource{}
	for _, res := range found {
		if res.GetKind() == "Secret" {
			result = append(result, *res)
			p.debug("Removing %s", res.CurId())
			err := rm.Remove(res.CurId())
			if err != nil {
				return nil, err
			}
		}
	}
	p.debug("extractAndRemoveSecrets end")
	return result, nil
}

func parseKey(r io.Reader) (*rsa.PublicKey, error) {
	// From main.go in github.com/bitnami-labs/sealed-secrets
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	certs, err := cert.ParseCertsPEM(data)
	if err != nil {
		return nil, err
	}

	// ParseCertsPem returns error if len(certs) == 0, but best to be sure...
	if len(certs) == 0 {
		return nil, errors.New("Failed to read any certificates")
	}

	cert, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Expected RSA public key but found %v", certs[0].PublicKey)
	}

	return cert, nil
}

func openCertURI(uri string) (io.ReadCloser, error) {
	// From main.go in github.com/bitnami-labs/sealed-secrets
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))
	c := &http.Client{Transport: t}

	resp, err := c.Get(uri)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot fetch %s", uri)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot fetch %q: %s", uri, resp.Status)
	}
	return resp.Body, nil
}

func isFilename(name string) (bool, error) {
	// From main.go in github.com/bitnami-labs/sealed-secrets
	u, err := url.Parse(name)
	if err != nil {
		return false, err
	}
	return u.Scheme == "", nil
}

func openCertLocal(filenameOrURIEnvs string) (io.ReadCloser, error) {
	rxEnvVar := regexp.MustCompile("\\$[A-Z0-9_]+")
	filenameOrURI := string(rxEnvVar.ReplaceAllFunc([]byte(filenameOrURIEnvs), func(s []byte) []byte {
		name := string(s)[1:]
		return []byte(os.Getenv(name))
	}))
	// From main.go in github.com/bitnami-labs/sealed-secrets
	// detect if a certificate is a local file or an URI.
	if ok, err := isFilename(filenameOrURI); err != nil {
		return nil, err
	} else if ok {
		return os.Open(filenameOrURI)
	}
	return openCertURI(filenameOrURI)
}

func (p *plugin) debug(format string, a ...interface{}) {
	if p.Verbose {
		fmt.Printf("Secret Sealer - "+format+"\n", a)
	}
}
