package main_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"strings"
	"testing"

	"github.com/pkg/errors"
	kusttest_test "sigs.k8s.io/kustomize/api/testutils/kusttest"
)

func TestSecretSealer_WithKubeSealInPath(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()
	tmp, err := ioutil.TempDir("/tmp/", "secretsealer.")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Error(err)
	}

	_, err = copyFile("../kubeseal_bin", path.Join(tmp, "kubeseal"))
	if err != nil {
		t.Error(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmp)
	defer os.Setenv("PATH", origPath)

	err = runAndAssert(th, "", true, true)

	if err != nil {
		t.Error(err)
	}
}

func TestSecretSealer_WithKubeSealInParent(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()

	_, err := copyFile("../kubeseal_bin", "../kubeseal")
	defer os.Remove("../kubeseal")
	if err != nil {
		t.Error(err)
	}

	const envName = "FIND_KUBESEAL"
	os.Unsetenv(envName)
	fmt.Print("before")
	err = runAndAssert(th,"", true, true)
	fmt.Println("after")
	fmt.Println(err)
	if err == nil {
		t.Error(errors.Errorf("Should have failed to find kubeseal but didn't"))
	}

	os.Setenv(envName, "1")
	defer os.Unsetenv(envName)
	err = runAndAssert(th, "", true, true)
	if err != nil {
		t.Error(err)
	}
}

func TestSecretSealer_WithTarget(t *testing.T) {
	th := kusttest_test.MakeEnhancedHarness(t).
		BuildGoPlugin("devjoes", "v1", "SecretSealer")
	defer th.Reset()
	tmp, err := ioutil.TempDir("/tmp/", "secretsealer.")
	defer os.RemoveAll(tmp)
	if err != nil {
		t.Error(err)
	}

	_, err = copyFile("../kubeseal_bin", path.Join(tmp, "kubeseal"))
	if err != nil {
		t.Error(err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmp)
	defer os.Setenv("PATH", origPath)

	var target = "namespace: foo"
	err = runAndAssert(th,target, false, true)
	target = "namespace: default"
	err = runAndAssert(th,target, true, true)

	if err != nil {
		t.Error(err)
	}
}

func runAndAssert(th *kusttest_test.HarnessEnhanced, options string, shouldReplace bool, shouldPass bool) error {
	fmt.Println("a")
	certPath, err := writeCert()
	if err != nil {
		return err
	}
	defer os.Remove(certPath)
	sealed, err := th.RunTransformer(
		fmt.Sprintf(`apiVersion: devjoes/v1
kind: SecretSealer
metadata:
  name: notImportantHere
cert: %s
%s
`, certPath, options),`
apiVersion: v1
data:
  foo: Zm9vCg==
  bar: YmFyCg==
kind: Secret
metadata:
  name: test
  namespace: default
`)

	expected := `apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  creationTimestamp: null
  name: test
  namespace: default
spec:
  encryptedData:
    bar: AgATyp1FXeqclIbsGN9Efc1cfJ9PBh+vF/RvGs6m2cYTvXVtptHW3rqW9DLeR7tjUICFZoR+9LLUPzHQu9Fokyj53jh2trmCf3kS96ggDBPdPUaN1I+AwwNxQ3MnxNI4mHnOJ1ZRuLdA5OvU/sPYeWZZ6gY0eMBK+tZavq8Ks4uFAO4CtHzgh7NslSios68Q7FUnlue+XePyQY+wD2SNIx54vE+fbFFP0f5LdVB8ZG6F+SSWSKV+2SiXbuWW8hA9OJ2HjueqNEU4j3CIQYRKZxlNfCOWyPuf37F/7HexpiRbVMU+FLXVGoeNocYz6x5JjXSaLegLVRlnvhdcxqvr0540FtD+hvfJGHECASIEViOaz+57VH4gKoJ4CO6A0DOtinOLMjr0gwQtb1QtIZvXTauohQaiKDLAbCNyeagfdGtMjYdhPGPWHC1FCfmzmRpJBhYtwh5iVoBhHed4Zqbv6CzHpT9kxxllk86A3X9lPD3cGK3OtOey0F5Zaq7vZM702suxVYBQ/H5EhNJRaWWk5FXTIlqeS3S7xViI7wmj+cQ3yEG/IxFvLRJ9wybToS0sQ+qf1QM64uwO0bj9Y0CnqE6d0+jy+R+x6rLiKBHx6MsGAWcxAnes3Ryz+FI0pi5G7LoiYwwXIBSpwYMfqAXZaedrw1ruUsTO3djApCCXTDR7hmTrfYlePe5/NhVdmV28TI6Ffy4X
    foo: AgCuaxi9qq8sMjUXrOBHGsSHd9RC4g0AlMLJhyons7KG1f2CVR5/gWZzX+1Rk0Gj9zmxqiiPndf5XFm5zkYx6bnSG5pTXczPvVKcDrU1NrE/Uhit0vw0uLLeQTVJ+JHKzJ7s0c3O5nsRIH3LndFA5c5sCds7UaBSeBoSkdiKytarzjnxraKMV0mWI9x2Eo8KUcVCeqRac7bZXiZvXJELDkS88rQEwkS29MOYv+cUxkAvpulTvsBI3aArrpu6KmGF+j8ZAAHvlywf6REmamuvlLigSSlv2aJL+vKTclzUGR/mXsRsq4q+cUM73cJRbkOJDvZ561XZ2zp6f80MPCGksTAK4JFCt3Z+Vz9HqGjafCWvKEEEptAOm5/lr5nQbJWBiX9lpw6sMU9JHOCw5mgzychAYl8vPoqT+1zWrd4tv4OnQLXSsOXu9nEzBFYJ/O/OFPj1j2oSToPFjALjlVc8XSoX7omGpepV/yAot86c4aZMtj+sBgnOR4vZAOStqzjGAA5ngYWUeT8sCFPGmh6hXPuMRgD9YnsH0+gXS6FPed2pnTnlOpIb+7Jebf0wRg4fBwMR0YrqgOESoGHHgVJ+ok9D++PbWQBP6UG8Dtd32tP46HQVLs19jwck0DXQm7yLIXUBbPDC+aj05J+v1wyRhVhiSCJFO8/gyosC6km0h9D3uug6jTGWqywi29fXRyBQAKUOT0lV
  template:
    metadata:
      creationTimestamp: null
      name: test
      namespace: default
status: {}`

	if err != nil {
		return err
	}
	bYaml, err := sealed.AsYaml()
	if err != nil {
		return err
	}

	// Kubeseal seems to use a different nonce every time so the encrypted lines will never actually match
	cleanYaml, removedLines := removeLines(string(bYaml), []string{"bar:", "foo:"})
	cleanExpected, expectedRemovedLines := removeLines(expected, []string{"bar:", "foo:"})

	if cleanYaml != cleanExpected {
		return errors.Errorf("Expected:\n%sGot:\n%s\n", cleanExpected, cleanYaml)
	}
	if removedLines != 2 || expectedRemovedLines != 2 {
		return errors.Errorf("Expected %d keys, found %d", expectedRemovedLines, removedLines)
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
