package main_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
	kusttest_test "sigs.k8s.io/kustomize/api/testutils/kusttest"
)

func TestSecretSealer_RequiresCert(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()
	tmp, err := ioutil.TempDir("/tmp/", "secretsealer.")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Error(err)
	}
	_, err = th.RunTransformer(
		`apiVersion: devjoes/v1
kind: SecretSealer
metadata:
  name: notImportantHere
`, secretSealerYaml)
	if err == nil || err.Error() != "Cert option is required" {
		println(err)
		t.Error("Doesn't error on missing cert option")
	}

	_, err = th.RunTransformer(
		`apiVersion: devjoes/v1
kind: SecretSealer
metadata:
  name: notImportantHere
  cert: /idontexist
`, secretSealerYaml)
	if err == nil {
		t.Error("Doesn't error if cert doesn't exist")
	}

}

func TestSecretSealer(t *testing.T) {
	os.Setenv("SESSION_KEY_SEED", "RlTttySb585amdle9tN3cz0XD2qChRmcbefSgwqudOYuhgKMfOjQDIKWovmNQkm")
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()

	err := runAndAssert(th, "", true, true, "")

	if err != nil {
		t.Error(err)
	}
}

func TestSecretSealerWithEnvVar(t *testing.T) {
	os.Setenv("SESSION_KEY_SEED", "RlTttySb585amdle9tN3cz0XD2qChRmcbefSgwqudOYuhgKMfOjQDIKWovmNQkm")
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()

	certPath, _ := writeCert()
	os.Setenv("FOO", "/tmp")
	certPath = "$FOO" + certPath[4:]
	err := runAndAssert(th, "", true, true, certPath)

	if err != nil {
		t.Error(err)
	}
	certPath, _ = writeCert()
	certPath = "$IDONTEXIT" + certPath
	err = runAndAssert(th, "", true, true, certPath)

	if err != nil {
		t.Error(err)
	}
}

func TestSecretSealer_WithTarget(t *testing.T) {
	os.Setenv("SESSION_KEY_SEED", "RlTttySb585amdle9tN3cz0XD2qChRmcbefSgwqudOYuhgKMfOjQDIKWovmNQkm")
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()
	tmp, err := ioutil.TempDir("/tmp/", "secretsealer.")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Error(err)
	}

	var target = "namespace: foo"
	err = runAndAssert(th, target, false, true, "")
	target = "namespace: default"
	err = runAndAssert(th, target, true, true, "")

	if err != nil {
		t.Error(err)
	}
}

func TestSecretSealer_WithoutSeed(t *testing.T) {
	os.Unsetenv("SESSION_KEY_SEED")
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()
	tmp, err := ioutil.TempDir("/tmp/", "secretsealer.")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Error(err)
	}

	err = runAndAssert(th, "namespace: default", true, false, "")
	if err != nil {
		t.Error(err)
	}
}

func runAndAssert(th *kusttest_test.HarnessEnhanced, options string, shouldReplace bool, shouldPass bool, certPath string) error {
	if certPath == "" {
		var err error
		certPath, err = writeCert()
		if err != nil {
			return err
		}
	}
	defer os.Remove(certPath)
	sealed, err := th.RunTransformer(
		fmt.Sprintf(`apiVersion: devjoes/v1
kind: SecretSealer
metadata:
  name: notImportantHere
cert: %s
%s
`, certPath, options), secretSealerYaml)

	yamlExpected :=
		`apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test
  namespace: default
spec:
  encryptedData:
	bar: AgAXEGhwpQVYBg7VpoEjHzFiw/gkSNdfwLhK1qAJ0pn9ir2Z/dJj98t8DnVdb6wCZo1t2DdGJqsQsGlmHqI6EbB2kAfVR3qhvP7EzQ2CvOmbnlQ+qErJ2Qz+lD3TzJS4UYkm8rZ5g86OduONEdkrcEjoI58OwvhFNTTQb8o7Bu4hAKJTNa0atJwh0fnSI22RpucWXLeyr+qeyjoqtHPu9O6Lr9NLIdqn1NIcVslHc9K7olpYF67ekBsmU2ZymoIZpKeF7CYyIG7y6v2NFb6Y8BSrwxcvG5kS91P5sj6XXeEZFrIvLB9Lz/usyql0jF/GFwZok9PpDp3xuV5vwPnMjHNqiFYGqk9AOdp0DGIHownLjjrOOU2DbWbeGzB+HV0jPBS83uziuuQO/FEqB/mbjdDSBEuJICAiV02TXU2uXsFKT+XiFgClXkvkEAqaOjCQRzTaRWdwz1tE+39fKUK/fOFNYIcUl3E354ExbvL4yp60K98Bblyya+KKXBDoTkI8Wu3R30DXmBtwllvBltCYi1e1CYDCd40lNpVMZBzkaJCgG1BkcoymbjyTtjjqiUPFoY2oMV9HoKoAQU0qDASmgsMSfJ5ssHLGP4/AJN+Mg3rNL8OmvyAZS0Ku9SdIJJ7pbSF4U2gn1mjARjx/iIcDq3SjCwme3ki9MP5lUTQh9IMSFWlJnDCqbsh40ZKsC0rsR5k9hCu4
	foo: AgAXEGhwpQVYBg7VpoEjHzFiw/gkSNdfwLhK1qAJ0pn9ir2Z/dJj98t8DnVdb6wCZo1t2DdGJqsQsGlmHqI6EbB2kAfVR3qhvP7EzQ2CvOmbnlQ+qErJ2Qz+lD3TzJS4UYkm8rZ5g86OduONEdkrcEjoI58OwvhFNTTQb8o7Bu4hAKJTNa0atJwh0fnSI22RpucWXLeyr+qeyjoqtHPu9O6Lr9NLIdqn1NIcVslHc9K7olpYF67ekBsmU2ZymoIZpKeF7CYyIG7y6v2NFb6Y8BSrwxcvG5kS91P5sj6XXeEZFrIvLB9Lz/usyql0jF/GFwZok9PpDp3xuV5vwPnMjHNqiFYGqk9AOdp0DGIHownLjjrOOU2DbWbeGzB+HV0jPBS83uziuuQO/FEqB/mbjdDSBEuJICAiV02TXU2uXsFKT+XiFgClXkvkEAqaOjCQRzTaRWdwz1tE+39fKUK/fOFNYIcUl3E354ExbvL4yp60K98Bblyya+KKXBDoTkI8Wu3R30DXmBtwllvBltCYi1e1CYDCd40lNpVMZBzkaJCgG1BkcoymbjyTtjjqiUPFoY2oMV9HoKoAQU0qDASmgsMSfJ5ssHLGP4/AJN+Mg3rNL8OmvyAZS0Ku9SdIJJ7pbSF4U2gn1mjARjx/iIcDq3SjCwme3ki9MP5lUTQh9IMSFW1HgTCHSiecxbIQdUNRzd41riSk
  template:
	metadata:
	  creationTimestamp: null
	  name: test
	  namespace: default
status: {}
`

	if err != nil {
		return err
	}
	bYaml, err := sealed.AsYaml()
	if err != nil {
		return err
	}
	yamlResult := strings.ReplaceAll(strings.ReplaceAll(string(bYaml), "\t", ""), " ", "")
	yamlExpected = strings.ReplaceAll(strings.ReplaceAll(string(yamlExpected), "\t", ""), " ", "")

	if yamlResult != yamlExpected && shouldPass {
		return errors.Errorf("\r\nExpected:\n%s\n\n\n\rGot:\n%s\n", yamlExpected, yamlResult)
	}

	return nil
}

func removeLines(input string, removeMatches []string) (string, int) {
	var builder strings.Builder
	var removed = 0
	for _, line := range strings.Split(input, "\n") {
		var skip bool = false
		for _, sub := range removeMatches {
			if strings.Contains(line, sub) {
				skip = true
			}
		}

		if !skip {
			builder.WriteString(fmt.Sprintf("%s\n", line))
		} else {
			removed++
		}
	}
	return strings.Trim(builder.String(), " \n\r"), removed
}

func copyFile(src, dst string) (int64, error) {
	// Why isn't this built in?!?
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	destination.Chmod(sourceFileStat.Mode())
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func writeCert() (string, error) {
	fCer, err := ioutil.TempFile("/tmp/", "cert.")
	if err != nil {
		return "", err
	}

	_, err = fCer.WriteString(testCert)
	if err != nil {
		return "", err
	}
	defer fCer.Close()
	return fCer.Name(), nil
}

const testCert = `
-----BEGIN CERTIFICATE-----
MIIErjCCApagAwIBAgIRAL7krIyy+2nR81cEvuh7JLwwDQYJKoZIhvcNAQELBQAw
ADAeFw0yMDAxMjcxNzAzMDRaFw0zMDAxMjQxNzAzMDRaMAAwggIiMA0GCSqGSIb3
DQEBAQUAA4ICDwAwggIKAoICAQC2YqpYDxEJIuNOAj/59CUkXQ3Pf86znidRUpZM
cPlfguHtI9poJaoYZhsmbwaP6xe6cfz8nDuheAgp95VMWlRMdqrATcDjWKOvRlcc
b033Z4R8aS02O9+w8KO4Do7uQ3jlnL2A8XiWWzlaOS816FE3qvDFOuXA6Mv8Kd82
+f3AcUx2vBOLLaqtoPY4FYGpaGFW0etWKr3wDrhkUSRKr1lfCXGfA+HwThxL+50R
eEYjwgNDGr0gVovg+OU2JfdSfMrFee5Jh28ZQf6fv9WtJsOFcfgPTuRAdKmEzcMi
utvF+tqCRskzaNZEW0Wd5c+whSbi5yfHJKT0i/cxxuz2jqm3cWRlpTFfBvIvdggf
gwSW/AqY73W7lb4RBjLWcMpaX7eHkZtAorh3xIsctwaeBH1xIr1erTlph8hsSNrB
B8nlgea6U8/o3qvjtQ8fNE5Qigisd48s8GAghw+gW3kNHfItFh0ksP/ju8vq3ZBV
fbkOIYvnVqj547fhulFnvmfl5/aVNxz8cljGPXB2ldb9+6UgPC2Rlelgpi9hRT6l
5uJbWnEOEZmElRuSDMGcVCYyJa0ya2vvqkoq01RbAsJfag2rUNAhwNgjtsz29OLj
GFh425Y3oodRV53Yo1ENanC/Oor1n1jMfpz+jSV5fDXz15L3UeHhUTaE/X1MJXaz
Zk0DpwIDAQABoyMwITAOBgNVHQ8BAf8EBAMCAAEwDwYDVR0TAQH/BAUwAwEB/zAN
BgkqhkiG9w0BAQsFAAOCAgEAqRXW9GgL5sVN/fCyx5AfmIQMrxakw/o6DvdCHrb8
oltzIr/wemlWRy0HRupKnlHFbSzusZ/4LfVubIrY4ImJclMUD75+u2JakWdNQew1
GbCvQ/21NBOsQQUnki/+oZczZD6T6bQYJ1Uia757LUpyVhP5H1wbze/z+hVAc4Zo
UCu+gCzCdbGoKxLaPfvHjOSg1dq3+9sMcU4EgH+MjHo1HMc+j6TGo7P9lrc6pA4k
45TkbfPGYk/N5t/Z+U6OKM6eC4tjZJ2sGEAbkwZ6tOvIzeKZkiCMRtr5mSSEscQu
POmlMt5fwF893BHa7xlSVhlSraayuwk6SYXsEk794zmBZ7CRRRWnZq95dHoITtM2
MLert8+hIcbNIsuFin/mMCZ5Yv2kGiE1+IttG4aL5x+BaQtXdkCdP31JCkMgotbZ
zbflwD7lnKr/X7m0rVKZ1ba89FanYc23m3q2M6vkBrfpBcqXd0If0XzqObpvkCQY
lRXcW2xC4bYBrkz9FMkdaXGe8qfndfzopZzuq4WhYFXUs6mQqUtD8RYdlz7SkbvN
CgS5nJsv/BPrkYSMSGQ9HrVHy7qST5oIoaiCsWRgT4cAdWynoMGp/Hwzs+B/zTV/
n5sJflx9u43DD0h+rYRXRE8qYzOGHu/HIJgWKnacAMwHOW70/mJIEs6MMP0T347J
3GM=
-----END CERTIFICATE-----
`

const secretSealerYaml = `
apiVersion: v1
data:
  foo: Zm9vCg==
  bar: YmFyCg==
kind: Secret
metadata:
  name: test
  namespace: default
`
