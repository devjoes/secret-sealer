package main

type SealedSecret struct {
	Name string
}

func (s *SealedSecret) ToYaml()(string, error) {
	return "foo" ,nil
}