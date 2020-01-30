package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	pluginHelper   *resmap.PluginHelpers
	Target         types.Selector `json:"target,omitempty" yaml:"target,omitempty"`
	Cert           string         `json:"cert,omitempty" yaml:"cert,omitempty"`
	//TODO: Add args for other kubeseal options
}

var KustomizePlugin plugin

func (p *plugin) Config(ph *resmap.PluginHelpers, c []byte) (err error) {
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
		return err
	}
	fmt.Print(string(c))
	return err
}

func (p *plugin) Transform(rm resmap.ResMap) error {
	secrets, err := p.extractAndRemoveSecrets(rm, p.Target)
	if err != nil {
		return err
	}
	kubeseal, err := NewKubesealAdapter(p.Cert)
	if err != nil {
		return err
	}

	for _, res := range secrets {
		if res.GetKind() == "Secret" {
			sSecret, err := p.sealSecret(&res, kubeseal)
			if err != nil {
				return errors.Wrapf(err, "An error occured whilst processing %s", res.OrgId())
			}
			rm.Append(&sSecret)
		}
	}

	return nil
}

func (p *plugin) sealSecret(secret *resource.Resource, kubeseal *kubesealAdapter) (resource.Resource, error) {
	secretYaml, err := secret.AsYAML()
	if err != nil {
		return resource.Resource{}, err
	}
	sealedYaml, err := kubeseal.Execute(secretYaml)
	if err != nil {
		return resource.Resource{}, err
	}
	sSecret, err := p.pluginHelper.ResmapFactory().NewResMapFromBytes(sealedYaml)
	if err != nil {
		return resource.Resource{}, err
	}
	if sSecret.Size() != 1 {
		return resource.Resource{}, errors.New(fmt.Sprintf("Expected a single SealedSecret but received %d", sSecret.Size()))
	}
	return *sSecret.GetByIndex(0), nil
}

func (p *plugin) extractAndRemoveSecrets(rm resmap.ResMap, selector types.Selector) ([]resource.Resource, error) {
	found, err := rm.Select(selector)
	if err != nil {
		return nil, err
	}

	var result = []resource.Resource{}
	for _, res := range found {
		if res.GetKind() == "Secret" {
			result = append(result, *res)
			err := rm.Remove(res.OrgId())
			if err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

type kubesealAdapter struct {
	binLocation string
	cert        string
}

func NewKubesealAdapter(cert string) (*kubesealAdapter, error) {
	//TODO: Ideally this should call kubeseal's package instead of invoking the binary
	fmt.Printf("\ncert = %s\n\n", cert)
	ksa := new(kubesealAdapter)
	ksPath, err := exec.LookPath("kubeseal")
	if err != nil {
		if os.Getenv("FIND_KUBESEAL") == "" {
			return nil, err
		}
		curDir, _ := os.Getwd()
		ksPath = findExecutable(curDir, "kubeseal")

		if ksPath == "" {
			return nil, err
		}
	}
	fmt.Printf("found %s\n", ksPath)
	ksa.binLocation = ksPath
	ksa.cert = cert

	return ksa, nil
}

func (o *kubesealAdapter) Execute(secretYaml []byte) ([]byte, error) {
	args := []string{"-o", "yaml"}
	if o.cert != "" {
		args = append(args, "--cert", o.cert)
	}
	fmt.Printf("%s %s", o.binLocation, strings.Join(args, " "))
	kubeseal := exec.Command(o.binLocation, args...)
	kubeseal.Stdin = bytes.NewReader(secretYaml)
	kubeseal.Stderr = os.Stderr
	output, err := kubeseal.Output()
	if err != nil {
		return nil, errors.Wrapf(err, "The command '%s %s' resulted %s",
			o.binLocation, strings.Join(args, " "), err.Error())
	}

	return output, nil
}

func findExecutable(dir string, name string) string {
	fmt.Printf("\nTrying %s\n", dir)
	p := path.Join(dir, name)
	_, err := os.Stat(p)
	if err == nil {
		// We assume that it is executable
		return path.Join(dir, name)
	}
	parent := filepath.Dir(dir)
	if len(parent) > 2 {
		return findExecutable(parent, name)
	}
	return ""
}
