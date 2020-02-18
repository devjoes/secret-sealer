#!/bin/bash

ok=0
kustomize build . --enable_alpha_plugins && ok=1 && echo "Deployed" | tee /tmp/log

if [ "$ok" != "1" ]; then
    printf "\033[0;31m FAILED: Kustomize errored\n\n"
    exit 1
fi

if grep -i "kind: secret" /tmp/log; then
    printf "\033[0;31m FAILED: Secrets detected\n\n"
    exit 2
fi

echo "Testing without seed"
kustomize build . --enable_alpha_plugins -o /tmp/without


export SESSION_KEY_SEED=jdjsjhsjdjkdskdsjddskjsdjkdsjkdsjksdjksdds
echo "Testing with seed"
kustomize build . --enable_alpha_plugins -o /tmp/with1
kustomize build . --enable_alpha_plugins -o /tmp/with2
export SESSION_KEY_SEED=

if diff /tmp/without /tmp/with1 > /dev/null; then
    printf "\033[0;31m FAILED: With seed and without seed match\n\n"
    exit 3
fi

if ! diff /tmp/with1 /tmp/with2 > /dev/null; then
    printf "\033[0;31m FAILED: With seed identical deployments dont match\n\n"
    exit 4
fi

printf "\033[0;32m \nAll tests pass!\n\n\033[0m;"