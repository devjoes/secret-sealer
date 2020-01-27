package main

import (
	//"strings"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	resmapFact	*resmap.Factory
	loader		ifc.Loader
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

var KustomizePlugin plugin

func (p *plugin) Config (loader ifc.Loader, resmapFact *resmap.Factory, buf []byte) error {
		p.loader = loader
		p.resmapFact = resmapFact
		return yaml.Unmarshal(buf, p)
	}


func (p *plugin) Generate() (resmap.ResMap, error) {
	ss := SealedSecret{}
	yaml, err := ss.ToYaml()
	if err != nil{
		return nil, err
	}
	return p.resmapFact.NewResMapFromBytes([]byte(yaml))
}