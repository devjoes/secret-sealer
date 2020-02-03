# Secret sealer

This plugin finds secrets and runs [Bitnami's Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) on them.

You can then declare the plugin like this:

     apiVersion: devjoes/v1
     kind: SecretSealer
     metadata:
       name: SecretSealer
     cert: http://example.com/my/public/key.pub

This will seal all available secrets. You can also add a selector such as "namespace:" to restrict which secrets get sealed. One thing to remember is that Sealed Secret's controller will auto rotate it's key every 30 days (customizable). So the cert property should point to a live copy of the key which you can expose using an ingress. Whilst this is just a public key if you want you can protect it with basic authentication and use a URL like user@password/my/public/key.pub. (Remmber this will get checked in to source control though.)

## Session key seed

Sealed Secrets uses a random session key each time it encrypts something (as you should). However if you are using something like [Flux](https://www.weave.works/oss/flux/) then this means that every time you run kustomize you will trigger a new deployment. So I have forked Sealed Secrets [here](https://github.com/devjoes/sealed-secrets/) and hobbled the encryption ever so slightly by basing the session key on a hash of the input and a seed that is provided in an environment variable. **This makes the encryption deterministic.** If you set SESSION_KEY_SEED to a 16 character password then it will enable this feature. If it is not set then this feature will be disabled.