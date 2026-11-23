# KServe

KServe is a Kubernetes-native model serving platform. You declare a model in
one `InferenceService` resource, and KServe turns it into a versioned,
autoscaled HTTP/gRPC inference endpoint, so teams don't have to hand-build a
serving stack for each model.

## What problem it solves

Getting a trained model into production on Kubernetes usually means a lot of
glue work for every model: a Deployment, a Service, an Ingress, an HPA, a
sidecar or init container that pulls weights from S3/GCS, health probes,
request logging, a canary rollout process, and a server image that speaks some
inference protocol. Each team builds this slightly differently. The result is
inconsistent APIs, idle GPUs, and nobody owning the platform.

KServe (it started as KFServing in Kubeflow and is now a CNCF project) makes
that a platform concern. You describe *what* to serve: the model format, where
the artifacts live, and how much compute it gets. KServe's controller handles
*how*: it pulls the model from storage, picks a matching **ServingRuntime**
(Triton, vLLM-backed Hugging Face, scikit-learn/XGBoost, LightGBM, TorchServe,
MLServer, or your own), wires up networking, and scales the pods. That includes
scaling to zero if you run the Knative-based serverless mode. Clients call a
standard API: the V1/V2 ("Open Inference Protocol") REST/gRPC endpoints for
classic models, and OpenAI-compatible `/openai/v1/...` routes for LLMs.

## Primary use cases and when to adopt it

- **A shared ML serving platform.** Platform or MLOps teams give data
  scientists and AI engineers a self-service path where "ship a model" means
  "open a PR with a YAML file". It fits naturally with GitOps (Argo CD, Flux).
- **Self-hosted LLM serving on Kubernetes.** The Hugging Face runtime runs vLLM
  under the hood. You can serve Llama, Qwen, Mistral, and other open-weight
  models behind an OpenAI-compatible API while KServe handles GPU scheduling,
  model caching, and autoscaling. Newer releases add an `LLMInferenceService`
  resource (built with llm-d) for features like disaggregated prefill/decode
  and KV-cache-aware routing.
- **Many models, uneven traffic.** Serverless mode scales idle models to zero,
  and ModelMesh / multi-model serving packs many small models onto shared pods.
  Both matter when you run hundreds of per-tenant or per-region models.
- **Safe model rollouts.** Canary traffic splitting between model revisions,
  plus pre/post-processing **Transformers** and **InferenceGraphs** (chains,
  ensembles, A/B splits) declared next to the predictor.

**Adopt it when** you already run Kubernetes, you serve more than a handful of
models, and you want one consistent serving API with built-in observability.
**Skip it when** you have one or two models and a managed service (SageMaker,
Vertex AI, Bedrock, a hosted LLM API) works for you, or when you don't run
Kubernetes. KServe adds operational weight: CRDs, cert-manager, an ingress or
Gateway, and optionally Knative and Istio.

## Getting started

### 1. Install KServe and deploy your first model

For a local or dev cluster (kind, minikube, k3d), the project provides a
quick-install script. Use the release branch that matches the
[docs](https://kserve.github.io/website/) for your version:

```bash
# Dev/test only: installs cert-manager, a networking layer, and KServe
curl -s "https://raw.githubusercontent.com/kserve/kserve/release-0.15/hack/quick_install.sh" | bash
```

Then declare a model. This one is a scikit-learn iris classifier from the
project's public example bucket:

```yaml
# sklearn-iris.yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: sklearn-iris
  namespace: models
spec:
  predictor:
    minReplicas: 1
    maxReplicas: 4
    model:
      modelFormat:
        name: sklearn
      storageUri: gs://kfserving-examples/models/sklearn/1.0/model
      resources:
        requests: { cpu: "100m", memory: "256Mi" }
        limits:   { cpu: "1",    memory: "1Gi" }
```

```bash
kubectl create namespace models
kubectl apply -f sklearn-iris.yaml
kubectl get inferenceservice sklearn-iris -n models
# NAME           URL                                       READY ...
# sklearn-iris   http://sklearn-iris.models.example.com    True
```

You never chose a container image. KServe matched `modelFormat: sklearn` to an
installed `ClusterServingRuntime`, and a storage initializer downloaded the
model before the server started.

### 2. Call the endpoint

KServe routes by `Host` header through your ingress or gateway:

```bash
SERVICE_HOSTNAME=$(kubectl get inferenceservice sklearn-iris -n models \
  -o jsonpath='{.status.url}' | cut -d/ -f3)

# For a local cluster, port-forward the ingress gateway, e.g.:
#   kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80
curl -s -H "Host: ${SERVICE_HOSTNAME}" -H "Content-Type: application/json" \
  http://localhost:8080/v1/models/sklearn-iris:predict \
  -d '{"instances": [[6.8, 2.8, 4.8, 1.4], [6.0, 3.4, 4.5, 1.6]]}'
# {"predictions":[1,1]}
```

The same server also answers the V2 / Open Inference Protocol
(`/v2/models/sklearn-iris/infer`), which gives clients one request shape across
Triton, MLServer, and the other runtimes.

### 3. Serve an LLM with an OpenAI-compatible API, then canary a new version

```yaml
apiVersion: serving.kserve.io/v1beta1
kind: InferenceService
metadata:
  name: llama3-chat
  namespace: models
spec:
  predictor:
    minReplicas: 1
    model:
      modelFormat:
        name: huggingface          # Hugging Face runtime, vLLM backend for supported models
      args:
        - --model_name=llama3
        - --model_id=meta-llama/Meta-Llama-3-8B-Instruct
        - --max-model-len=8192
      env:
        - name: HF_TOKEN
          valueFrom:
            secretKeyRef: { name: hf-secret, key: HF_TOKEN }
      resources:
        requests: { cpu: "4", memory: "24Gi", nvidia.com/gpu: "1" }
        limits:   { cpu: "8", memory: "32Gi", nvidia.com/gpu: "1" }
```

```bash
curl -s -H "Host: ${LLM_HOSTNAME}" -H "Content-Type: application/json" \
  http://localhost:8080/openai/v1/chat/completions \
  -d '{"model":"llama3","messages":[{"role":"user","content":"Explain KServe in one line"}]}'
```

Any OpenAI SDK works against this route if you point `base_url` at
`.../openai/v1`.

To roll out a new model version safely in serverless mode, change the spec and
set a canary percentage. KServe keeps the last ready revision and splits
traffic between the two:

```yaml
spec:
  predictor:
    canaryTrafficPercent: 10      # 10% to the new revision, 90% to the previous one
    model:
      modelFormat: { name: sklearn }
      storageUri: gs://my-bucket/models/iris/v2
```

Watch error rates and latency. Then raise the percentage, or remove the field
to promote the new revision to 100%. Setting it back to `0` rolls back.

## Common pitfalls and things to watch out for

- **Choose a deployment mode deliberately.** *Serverless* (Knative) gives you
  scale-to-zero, revisions, and canary splits, but it requires Knative and a
  networking layer and adds cold-start latency. *Raw/Standard* mode (annotation
  `serving.kserve.io/deploymentMode: RawDeployment`, the default for LLM-focused
  installs in recent releases) uses plain Deployments and HPA/KEDA. It is
  simpler to operate but has no scale-to-zero, and `canaryTrafficPercent` does
  not apply. Many teams use Raw mode for GPU LLMs and Serverless for small CPU
  models.
- **Scale-to-zero plus large models means slow cold starts.** Pulling 16 GB of
  weights from S3 and loading them onto a GPU can take minutes, which breaks
  client timeouts. For anything user-facing, set `minReplicas: 1` or more, and
  use the model cache / `LocalModelCache` (or a PVC/OCI `storageUri`) so you
  don't download weights again on every pod start.
- **Pick the right autoscaling signal.** The default concurrency- or CPU-based
  scaling works poorly for LLMs: GPU utilization stays high even at low load,
  and a single long generation holds one slot for seconds. Scale on
  concurrency, queue depth, or tokens per second (through KEDA and custom
  metrics). Also set `containerConcurrency` so a pod isn't overloaded past what
  vLLM can batch.
- **Storage credentials and `storageUri` errors are the #1 "stuck not Ready"
  cause.** If the storage initializer can't reach the bucket, the pod sits in
  `Init` forever. Attach credentials through a ServiceAccount (IRSA or Workload
  Identity on cloud, or a secret annotated for KServe). Debug with
  `kubectl logs <pod> -c storage-initializer`.
- **Check the runtime and version matrix.** Your model's framework version must
  match what the runtime image supports. An XGBoost model saved with a newer
  library version can fail to load in an older runtime. Pin your
  `ServingRuntime` images, and when the built-in ones don't fit, run a custom
  runtime instead of hacking around them. KServe's APIs and install manifests
  also change between minor releases (deployment mode defaults, Gateway API
  support, the LLM resources), so read the release notes before upgrading.
- **Networking is where most install time goes.** Host-header routing, Istio or
  Gateway API configuration, DNS/`ingressDomain` settings, and TLS through
  cert-manager confuse everyone the first time. Decide early whether clients
  call through an external gateway or the cluster-local URL
  (`<name>.<ns>.svc.cluster.local`).
- **KServe serves models; it doesn't manage the ML lifecycle.** It has no model
  registry, no evaluation, and no feature store. Pair it with MLflow or a
  registry for versioned artifacts, CI that writes the new `storageUri` into
  Git, and observability (Prometheus metrics, OpenTelemetry, payload logging to
  Kafka or a sink) so you can see when a canary is actually worse.
