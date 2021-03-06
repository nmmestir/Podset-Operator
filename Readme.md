

# FeatureFlag Kubernetes AppConfig Operator

## Quickstart

Build:
```bash=
make build
```

Install CRD:
```bash=
make install
```

Run locally the controller:
```bash=
make run
```

Create Example
```yaml
cat << EOF | kubectl apply -f -
apiVersion: my.domain/v1alpha1
kind: PodSet
metadata:
  name: podset-sample
spec:
  ClientID: 'new'
  Application: 'PodSetTest'
  ClientConfigurationVersion: 1
  Configuration: 'configuration-pod-set1'
  Environment: 'dev'
  Labels:
    app: podset
EOF    
```

Install Go debugger

```bash
go get github.com/go-delve/delve/cmd/dlv
```

The Idea is to have a sample app based on eks-example, that will fetch AppConfig infos, and just print them in the output of the page

example of commands
```
$ aws appconfig get-configuration --application HelloWorld --environment TrueEnv --configuration TrueProfile --client-id me file.txt
{
    "ConfigurationVersion": "1",
    "ContentType": "text/plain"
}
$ cat file.txt
key1=value1
key2=value2
key3=value3
```   

ARN Service Account to access AppConfig 
arn:aws:iam::111492154997:policy/Appconfig-access

eksctl utils associate-iam-oidc-provider \
    --cluster bedrock-cluster \
    --approve

eksctl create iamserviceaccount \
    --name podset \
    --namespace hello-world \
    --cluster bedrock-cluster \
    --attach-policy-arn "arn:aws:iam::111492154997:policy/Appconfig-access" \
    --approve \
    --override-existing-serviceaccounts

  