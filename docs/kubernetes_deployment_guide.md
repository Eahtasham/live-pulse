# LivePulse on DigitalOcean Kubernetes (DOKS)

This document is both:

1) a **deployment plan** for running LivePulse on a DigitalOcean Kubernetes (DOKS) cluster, and
2) an **all-in-one guide** to the Kubernetes concepts you’ll touch (pods, deployments, services, ingress, storage, and autoscaling).

It assumes you want these components **inside the cluster**:

- `web` (Next.js)
- `api` (Go)
- `realtime` (Go / WebSockets)
- `redis` (Redis 7)
- `db` (PostgreSQL 16)

---

## 0) Can a 1–2 node (1 vCPU / 2 GiB) DOKS cluster handle this?

### Short, honest answer
- **1 node**: it will *run* for dev/demo/light traffic, but it’s **fragile** (no high availability) and **resource-tight**. Expect occasional CPU throttling and a real risk of OOM kills under load, especially with Next.js + Postgres.
- **2 nodes**: much more workable for dev/staging and small production *if traffic is low* and you use conservative resource requests.

### What makes it tight?
On small nodes, you are competing with:
- **Kubernetes “system” pods** (CNI, kube-proxy, CSI, DNS, etc.)
- an **ingress controller** (usually `ingress-nginx`)
- your 5 app components

The two biggest consumers are usually:
- **PostgreSQL** (memory)
- **Next.js** (memory spikes during SSR and in general Node runtime overhead)

### Practical minimum sizing (what I’d do)
If you want **everything in-cluster**, a safer baseline is:
- **2 nodes** minimum
- ideally **2 vCPU / 4 GiB** nodes for “small production”

If you must stay on **1 vCPU / 2 GiB** nodes, the most reliable approach is:
- keep **db + redis managed** (DigitalOcean Managed Databases)
- run only `web`, `api`, `realtime` in the cluster

You explicitly asked to run DB + Redis in-cluster, so this guide shows that — just treat it as **dev/staging-first** unless you upgrade the nodes.

### Suggested starting resource requests (for a tiny cluster)
These are conservative *requests* to keep scheduling possible on small nodes. Real usage may exceed them.

| Component | Kind | Replicas | CPU request | Memory request | Notes |
|---|---:|---:|---:|---:|---|
| `api` | Deployment | 1 | 50m | 128Mi | Go is efficient; add headroom if you enable heavy logging |
| `realtime` | Deployment | 1 | 50m | 128Mi | WebSockets: memory grows with connections |
| `web` | Deployment | 1 | 100m | 256Mi | Next.js often wants **512Mi+** depending on traffic |
| `redis` | StatefulSet (or Deployment) | 1 | 50m | 256Mi | turn on AOF only if you need it; it costs IO |
| `postgres` | StatefulSet | 1 | 100m | 512Mi | Postgres is happier at **1Gi+** for anything real |
| ingress | Deployment | 1 | 100m | 256Mi | `ingress-nginx` typical baseline |

If you go to 2 nodes, keep **replicas = 1** initially, then scale stateless services.

---

## 1) LivePulse architecture (Kubernetes view)

### Runtime topology

```
                    (public internet)
                           |
                           v
                    [LoadBalancer]
                           |
                           v
                      [Ingress]
                  /       |       \
                 /        |        \
                v         v         v
         web:3000     api:8080  realtime:8081 (ws)
                \         |         /
                 \        |        /
                  v       v       v
                       redis:6379
                           |
                           v
                      postgres:5432
```

### Kubernetes objects you’ll use
- `Namespace`: logical isolation (we’ll use one namespace, e.g. `livepulse`)
- `Deployment`: for stateless workloads (`web`, `api`, `realtime`)
- `StatefulSet`: for stateful workloads (`postgres`, optionally `redis`)
- `Service`:
  - `ClusterIP` for internal service discovery (e.g. `api`, `redis`, `postgres`)
- `Ingress`: external HTTP(S) routing to Services
- `Secret` / `ConfigMap`: configuration
- `PersistentVolumeClaim` (PVC): durable storage for Postgres / Redis
- `HorizontalPodAutoscaler` (HPA): autoscaling for Deployments

---

## 2) Kubernetes fundamentals (how it actually works)

### 2.1 Control plane and reconciliation
Kubernetes is a “desired state” system.

- You declare **what you want** (e.g. “2 replicas of api”).
- Controllers continuously **reconcile** reality to match that.

Example: A `Deployment` sets a desired number of replicas. A controller ensures a `ReplicaSet` exists, and the ReplicaSet ensures the right number of Pods exist.

If a Pod dies, Kubernetes doesn’t “restart the pod”. Instead, the ReplicaSet creates a **new** pod to replace it.

### 2.2 Pods, containers, scheduling
- A **Pod** is the smallest schedulable unit.
- A Pod can contain 1+ containers, but most apps use **1 container per Pod**.
- The **scheduler** assigns Pods to nodes based on:
  - CPU/memory **requests**
  - constraints (affinity, taints/tolerations)
  - available capacity

### 2.3 Requests vs limits (very important)
For each container you can set:
- **requests**: what the scheduler assumes you need
- **limits**: a hard cap

CPU:
- If you exceed your CPU limit, you get **throttled**.

Memory:
- If you exceed your memory limit, you get **OOMKilled**.

On tiny nodes, set requests realistically but not so high that nothing schedules.

### 2.4 Health probes: readiness vs liveness
- **readinessProbe**: “should this pod receive traffic?”
- **livenessProbe**: “should Kubernetes restart this container?”

For this repo:
- `api` and `realtime` expose `GET /healthz`.
- `web` can use `GET /` for a basic check.

---

## 3) Autoscaling explained (pods and nodes)

### 3.1 Deployment scaling (manual)
You can always scale a Deployment:

- `kubectl scale deployment api --replicas=2`

Kubernetes creates additional pods and services load-balance between them.

### 3.2 HPA (Horizontal Pod Autoscaler)
HPA automatically adjusts `spec.replicas` on a Deployment.

**How it works (CPU)**
- HPA reads CPU usage metrics.
- It targets a utilization percentage of the **requested CPU**.

Roughly:

$$
\text{desiredReplicas} = \left\lceil \text{currentReplicas} \times \frac{\text{currentMetric}}{\text{targetMetric}} \right\rceil
$$

**Key dependency:** metrics must exist.
- You usually need **metrics-server** installed.

**What can be used as HPA signals**
- CPU utilization (most common)
- memory utilization (possible, but often not a great autoscaling signal)
- custom metrics (requires Prometheus adapter or similar)

#### “Different metrics” in practice (autoscaling/v2)
The HPA API (`autoscaling/v2`) supports multiple metric types:

- **Resource**: CPU/memory (from metrics-server)
- **Pods**: a metric averaged across pods (typically via custom metrics pipeline)
- **Object**: a metric tied to a single Kubernetes object (e.g. requests per second on an Ingress)
- **External**: metrics from outside the cluster (queues, cloud services), often via KEDA

For this stack on DOKS:
- CPU-based HPA is the easiest win (install metrics-server)
- if you want to scale on **HTTP RPS**, **latency**, or **active WebSocket connections**, you typically add:
  - Prometheus (to scrape/export metrics), and
  - a metrics adapter (Prometheus Adapter) *or* KEDA (for event-driven scaling)

### 3.3 Cluster Autoscaler (nodes)
HPA adds pods; Cluster Autoscaler adds nodes.

If HPA scales up and pods become **Pending** due to no room, Cluster Autoscaler can increase node count (within your node pool min/max).

On DOKS, node autoscaling is typically configured on the node pool in the control panel.

### 3.4 Stateful workloads don’t autoscale like stateless ones
- Postgres and Redis generally do **not** scale horizontally via HPA.
- You can scale StatefulSets, but replication/failover is a database-level concern.

For production, consider managed databases.

---

## 4) DigitalOcean specifics you should know

### 4.1 Networking CIDRs
You provided:
- Node network: `10.122.0.0/20`
- Pod network: `10.108.0.0/16`
- Service network: `10.109.0.0/19`

These ranges **don’t overlap**, which is good.

What to watch for:
- don’t overlap with on-prem VPN ranges or other VPCs you plan to peer

### 4.2 Storage
For Postgres (and optionally Redis) you want a `PersistentVolumeClaim` using the cluster’s default StorageClass.

On DOKS this is typically backed by DigitalOcean Block Storage via CSI.

You should confirm your StorageClass name:
- `kubectl get storageclass`

### 4.3 Ingress and Load Balancers
Common pattern:
- install `ingress-nginx` (creates a Service of type `LoadBalancer`)
- create Ingress resources with host rules:
  - `livepulse.yourdomain.com` → web
  - `api.livepulse.yourdomain.com` → api
  - `rt.livepulse.yourdomain.com` → realtime (WebSockets)

TLS is usually via `cert-manager` + Let’s Encrypt.

---

## 5) Deployment plan (step-by-step)

### 5.1 Prerequisites on your machine
- `kubectl`
- `helm`
- optional: `doctl` (nice for registry and cluster interactions)

**Important:** your kubeconfig file is a credential. Don’t commit it. This repo now ignores common kubeconfig patterns via `.gitignore`.

### 5.2 Connect to the cluster
If you already have a kubeconfig file, you can point `kubectl` at it:

```bash
export KUBECONFIG=./k8s-livepulse-kubeconfig.yaml
kubectl get nodes
```

If `kubectl` can’t connect, verify your DO token / kubeconfig is valid.

### 5.3 Choose a namespace
Use one namespace to start:

```bash
kubectl create namespace livepulse
```

### 5.4 Install metrics-server (for HPA)
First check if it’s already there:

```bash
kubectl get deployment -n kube-system | grep -i metrics
```

If not installed, install using Helm (example; your org may prefer another method):

```bash
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server/
helm repo update
helm upgrade --install metrics-server metrics-server/metrics-server \
  --namespace kube-system
```

Validate:

```bash
kubectl top nodes
kubectl top pods -n kube-system
```

### 5.5 Install ingress-nginx

```bash
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace
```

Get the external IP:

```bash
kubectl get svc -n ingress-nginx
```

You’ll point DNS records at that load balancer.

### 5.6 (Optional) Install cert-manager for TLS

```bash
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set crds.enabled=true
```

### 5.7 Create Secrets and Config
Use `Secret` for sensitive values:
- `JWT_SECRET`
- `AUTH_SECRET`
- Postgres password

Use `ConfigMap` (or plain env vars) for non-sensitive configuration.

**Minimum env vars from this repo**
- API (`apps/api`)
  - `API_PORT=8080`
  - `DATABASE_URL=postgres://livepulse:<POSTGRES_PASSWORD>@postgres:5432/livepulse?sslmode=disable`
  - `REDIS_URL=redis://redis:6379/0`
  - `JWT_SECRET=...`
- Realtime (`apps/realtime`)
  - `REALTIME_PORT=8081`
  - `REDIS_URL=redis://redis:6379/0`
  - `API_BASE_URL=http://api:8080`
- Web (`apps/web`)
  - `PORT=3000`
  - `API_URL=http://api:8080` (server-side calls)
  - `NEXT_PUBLIC_API_URL=https://api.<domain>` (browser calls)
  - `NEXT_PUBLIC_WS_URL=wss://rt.<domain>`
  - `AUTH_URL=https://<domain>`
  - `AUTH_SECRET=...`

#### Important: `NEXT_PUBLIC_*` vars are effectively build-time for the browser
In Next.js, `NEXT_PUBLIC_*` variables are bundled into the client build.

That means:
- if you use a **prebuilt** `web` image, changing `NEXT_PUBLIC_API_URL` / `NEXT_PUBLIC_WS_URL` at container runtime may **not** change what the browser uses
- the safest approach is to **build/push your own `web` image** per environment/domain

If you want runtime-configurable browser URLs, you usually implement a small runtime config endpoint/script in Next.js and read it in the client — but that’s outside Kubernetes itself.

#### Creating secrets (recommended)
Prefer creating secrets via CLI (so you don’t accidentally commit them):

```bash
kubectl -n livepulse create secret generic livepulse-secrets \
  --from-literal=JWT_SECRET='<random>' \
  --from-literal=AUTH_SECRET='<random>' \
  --from-literal=POSTGRES_PASSWORD='<random>'
```

### 5.8 Deploy Redis
For a small cluster, a single-replica Redis is fine.

Key points:
- use a PVC if you want AOF persistence
- expose a ClusterIP Service `redis:6379`

### 5.9 Deploy Postgres
For Postgres you want:
- a PVC (block storage)
- a Service `postgres:5432`
- a StatefulSet with **1 replica**

### 5.10 Run migrations
This repo uses `golang-migrate` migrations in `db/migrations/`.

Two common options:

**Option A (simplest): port-forward and run migrate locally**

```bash
kubectl -n livepulse port-forward svc/postgres 5432:5432

# in another terminal, from repo root
migrate -path db/migrations \
  -database "postgres://livepulse:<PASSWORD>@localhost:5432/livepulse?sslmode=disable" up
```

**Option B: a one-off Kubernetes Job**
- build an image that contains the migration files
- run `migrate up` inside the cluster

Option A is usually fine for first-time setup.

### 5.11 Deploy api + realtime
Use Deployments with:
- `readinessProbe` and `livenessProbe` hitting `/healthz`
- resource requests/limits (tiny cluster-friendly)
- Services (`ClusterIP`) named `api` and `realtime`

### 5.12 Deploy web
Use a Deployment and a Service named `web`.

### 5.13 Expose via Ingress (host-based routing)
Suggested hostnames:
- `livepulse.<domain>` → `web`
- `api.livepulse.<domain>` → `api`
- `rt.livepulse.<domain>` → `realtime`

WebSockets:
- `ingress-nginx` supports them automatically; ensure timeouts are reasonable.

### 5.14 Add autoscaling (optional on tiny clusters)
Enable HPA for **stateless** services:
- `api`
- `realtime`
- `web`

On a 1–2 node tiny cluster, keep max replicas small (e.g. 2) or you’ll just starve the node.

### 5.15 Validate and operate
Useful commands:

```bash
kubectl -n livepulse get all
kubectl -n livepulse get pods -o wide
kubectl -n livepulse logs deploy/api
kubectl -n livepulse logs deploy/realtime
kubectl -n livepulse logs deploy/web
kubectl -n livepulse top pods
kubectl -n livepulse describe pod <pod-name>
```

---

## 6) What scaling will and won’t do for this stack

### 6.1 Scaling `api`
Works well.
- `Service` load-balances across pods.
- You may need to ensure DB connection pooling is tuned (too many pods can overload Postgres).

### 6.2 Scaling `realtime`
Works, with caveats.
- each websocket connection terminates at one pod
- Redis pub/sub ensures events get to the pods that have clients

For very large scale you’ll want to think about:
- sticky sessions (sometimes),
- ingress timeouts,
- per-pod connection limits.

### 6.3 Scaling `web`
Works.
- most requests are independent
- Node memory usage is the main limiter

### 6.4 Scaling Postgres and Redis
Not “HPA style”.
- keep them as single-replica in-cluster for simplicity
- for production, use managed offerings or a proper HA operator

---

## 7) Recommended rollout order (the safe way)

1) Ingress controller
2) Redis
3) Postgres + PVC
4) Run migrations
5) API
6) Realtime
7) Web
8) (Optional) HPA on stateless components

---

## 8) Production notes (if you go beyond demo)

If you want reliability on DO:
- strongly consider DigitalOcean **Managed Postgres** + **Managed Redis**
- set up backups and alerting
- run at least 2 nodes, ideally 3 for higher availability
- add `PodDisruptionBudgets` for stateless services
- use separate node pools (optional): one for stateful, one for stateless

---

## 9) Next step: generating actual Kubernetes manifests

This document intentionally explains the concepts first.

If you want, I can generate a ready-to-apply `k8s/` folder for this repo containing:
- `Namespace`
- `Secrets` + `ConfigMaps`
- `StatefulSet` + `PVC` + `Service` for Postgres and Redis
- `Deployment` + `Service` for api/realtime/web
- `Ingress` resources (with optional TLS)
- optional HPAs for api/realtime/web

Tell me:
1) your domain name(s) (or that you’ll use a single domain)
2) do you want TLS via Let’s Encrypt (`cert-manager`) or plain HTTP for now?
3) do you want to use GHCR images or build/push to DigitalOcean Container Registry?

---

## 10) Appendix: minimal manifest templates (copy/paste starting points)

These are intentionally minimal so you can understand the moving pieces. Replace image names, domains, and secrets to fit your setup.

### 10.1 Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: livepulse
```

### 10.2 Postgres (StatefulSet + Service + PVC)

Before applying, confirm your StorageClass:

```bash
kubectl get storageclass
```

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data
  namespace: livepulse
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 10Gi
  # storageClassName: do-block-storage
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: livepulse
spec:
  type: ClusterIP
  ports:
    - name: postgres
      port: 5432
      targetPort: 5432
  selector:
    app: postgres
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: livepulse
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
              name: postgres
          env:
            - name: POSTGRES_USER
              value: livepulse
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: livepulse-secrets
                  key: POSTGRES_PASSWORD
            - name: POSTGRES_DB
              value: livepulse
          volumeMounts:
            - name: data
              mountPath: /var/lib/postgresql/data
          resources:
            requests:
              cpu: 100m
              memory: 512Mi
            limits:
              cpu: 500m
              memory: 1Gi
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: postgres-data
```

### 10.3 Redis (single replica)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: redis
  namespace: livepulse
spec:
  type: ClusterIP
  ports:
    - name: redis
      port: 6379
      targetPort: 6379
  selector:
    app: redis
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: livepulse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
        - name: redis
          image: redis:7-alpine
          args: ["redis-server"]
          ports:
            - containerPort: 6379
              name: redis
          resources:
            requests:
              cpu: 50m
              memory: 256Mi
            limits:
              cpu: 250m
              memory: 512Mi
```

### 10.4 API (Deployment + Service)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: api
  namespace: livepulse
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8080
      targetPort: 8080
  selector:
    app: api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: livepulse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
        - name: api
          image: ghcr.io/eahtasham/live-pulse/api:latest
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: API_PORT
              value: "8080"
            - name: REDIS_URL
              value: redis://redis:6379/0
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: livepulse-secrets
                  key: POSTGRES_PASSWORD
            - name: DATABASE_URL
              value: postgres://livepulse:$(POSTGRES_PASSWORD)@postgres:5432/livepulse?sslmode=disable
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: livepulse-secrets
                  key: JWT_SECRET
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 3
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 20
          resources:
            requests:
              cpu: 50m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
```

### 10.5 Realtime (Deployment + Service)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: realtime
  namespace: livepulse
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8081
      targetPort: 8081
  selector:
    app: realtime
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: realtime
  namespace: livepulse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: realtime
  template:
    metadata:
      labels:
        app: realtime
    spec:
      containers:
        - name: realtime
          image: ghcr.io/eahtasham/live-pulse/realtime:latest
          ports:
            - containerPort: 8081
              name: http
          env:
            - name: REALTIME_PORT
              value: "8081"
            - name: REDIS_URL
              value: redis://redis:6379/0
            - name: API_BASE_URL
              value: http://api:8080
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8081
            initialDelaySeconds: 3
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8081
            initialDelaySeconds: 10
            periodSeconds: 20
          resources:
            requests:
              cpu: 50m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
```

### 10.6 Web (Deployment + Service)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: web
  namespace: livepulse
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 3000
      targetPort: 3000
  selector:
    app: web
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: livepulse
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: web
          image: ghcr.io/eahtasham/live-pulse/web:latest
          ports:
            - containerPort: 3000
              name: http
          env:
            - name: PORT
              value: "3000"
            - name: API_URL
              value: http://api:8080
            # note: NEXT_PUBLIC_* is bundled for browser usage; rebuild the web image if needed
            - name: NEXT_PUBLIC_API_URL
              value: https://api.livepulse.<domain>
            - name: NEXT_PUBLIC_WS_URL
              value: wss://rt.livepulse.<domain>
            - name: AUTH_URL
              value: https://livepulse.<domain>
            - name: AUTH_SECRET
              valueFrom:
                secretKeyRef:
                  name: livepulse-secrets
                  key: AUTH_SECRET
          readinessProbe:
            httpGet:
              path: /
              port: 3000
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              cpu: 100m
              memory: 256Mi
            limits:
              cpu: 500m
              memory: 512Mi
```

### 10.7 Ingress (3 hosts)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: livepulse
  namespace: livepulse
  annotations:
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  ingressClassName: nginx
  rules:
    - host: livepulse.<domain>
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web
                port:
                  number: 3000
    - host: api.livepulse.<domain>
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 8080
    - host: rt.livepulse.<domain>
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: realtime
                port:
                  number: 8081
  # tls:
  #   - hosts:
  #       - livepulse.<domain>
  #       - api.livepulse.<domain>
  #       - rt.livepulse.<domain>
  #     secretName: livepulse-tls
```

### 10.8 HPA example (CPU-based)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api
  namespace: livepulse
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 1
  maxReplicas: 2
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```
