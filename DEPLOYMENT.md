# ApplyFlow AWS deployment log

This is a chronological record of **production deployment** (not local Docker/LocalStack). Local setup remains: `docker compose` + `scripts/setup-local-aws.sh` + `make dev-api` / `make dev-worker`.

**Region:** ap-south-2 (Hyderabad)  
**When:** 12–13 September 2026

Do **not** commit `infra/ecs-task-api.json`, `infra/ecs-task-worker.json`, or `.env`. They contain the DB password, JWT secret, and Google client secret.

---

## Current state (end of this log)

| Piece | State |
|--------|--------|
| ECS API + worker | Desired count **0** (no Fargate hours) |
| Application Load Balancer | **Deleted** (was the main idle cost besides RDS) |
| Target group `applyflow-api-tg` | Still exists |
| RDS `applyflow-db` | **Available** (kept for data) |
| Bastion `applyflow-bastion` | **Stopped** |
| ECR images | Present (`:latest` and git SHA tags) |
| GitHub Actions CI/CD | Working (OIDC → ECR → register task def → update service) |
| Public HTTPS / Google login on AWS | **Not done** (see troubles) |
| Frontend on AWS | **Not done** |

Idle cost while RDS stays on a free-plan `db.t4g.micro` is expected to be ~$0 during the 6-month trial, if Billing → Free Tier agrees. Turning ALB + Fargate on 24/7 is what would create a real bill.

---

## 1. AWS CLI and account

Used a configured AWS CLI profile for deployment.

```bash
export AWS_PROFILE=<your-profile>
export AWS_REGION=ap-south-2
```

---

## 2. Object storage and queue

| Resource | Value |
|----------|--------|
| S3 bucket | `applyflow-uploads` (created with CLI; `ap-south-2` needs a LocationConstraint) |
| SQS | `applyflow-email-queue` |
| SQS visibility | 120 seconds |

**Trouble:** App `EnsureBucket` in `ap-south-2` failed without LocationConstraint.  
**Fix:** Create the bucket with AWS CLI instead of relying on the first app start.

Local `.env` still uses LocalStack (`AWS_ENDPOINT_URL=http://localhost:4566`). Production ECS task defs **omit** `AWS_ENDPOINT_URL` so the SDK talks to real AWS.

---

## 3. Network (existing VPC)

Default VPC in `ap-south-2`:

| Resource | ID |
|----------|-----|
| VPC | Default VPC |
| Subnet 2a (public, bastion + ECS) | Public subnet |
| Subnet 2b | Public subnet |
| Subnet 2c | Public subnet |

---

## 4. RDS Postgres (private)

| Field | Value |
|--------|--------|
| Identifier | `applyflow-db` |
| Engine | Postgres 16 |
| Class | `db.t4g.micro`, 20 GB gp2, Single-AZ |
| DB name / user | `applyflow` / `applyflow` |
| Public access | **No** |
| RDS security group | Private security group (5432 from bastion + ECS SG) |

**Trouble:** Instance was created without database `applyflow`.  
**Fix:** `CREATE DATABASE applyflow` over a tunnel.

Laptop access is SSH tunnel through the bastion (not public RDS):

```text
ssh -i ~/.ssh/applyflow-bastion.pem -L 5433:<rds-endpoint>:5432 -N ec2-user@<bastion-public-ip>
```

Then `DATABASE_URL` on the laptop uses `localhost:5433`. ECS tasks use the RDS hostname on port **5432** (not 5433).

---

## 5. Bastion EC2

| Field | Value |
|--------|--------|
| Name | `applyflow-bastion` |
| Type | `t3.micro` |
| Key | `~/.ssh/applyflow-bastion.pem` |
| State at end of log | **stopped** |

The bastion is only for reaching the private database via SSH tunnel. It is stopped when not in use to avoid costs.

---

## 6. Container images (ECR)

| Repo | URI |
|------|-----|
| API | `applyflow-api` |
| Worker | `applyflow-worker` |

`Dockerfile` uses `--build-arg SERVICE=api` or `worker` and `GOOS=linux`. Images were built as **`linux/amd64`** for Fargate.

---

## 7. ECS Fargate

| Resource | Value |
|----------|--------|
| Cluster | `applyflow` |
| Services | `applyflow-api`, `applyflow-worker` |
| Launch type | Fargate, `cpu 256` / `memory 512` |
| API container | name `api`, port 8080 |
| Worker container | name `worker` |
| Assign public IP | ENABLED (needed for ECR pull + outbound in this subnet setup) |
| Execution role | `applyflow-ecs-execution` + `AmazonECSTaskExecutionRolePolicy` |
| Task role | `applyflow-ecs-task` + inline policy `applyflow-s3-sqs` (`infra/applyflow-task-policy.json`) |
| Logs | CloudWatch group `/ecs/applyflow`, prefixes `api` and `worker` |
| ECS tasks SG | Security group for port 8080 (later also from ALB SG) |

Task definition JSON lives only on disk (`infra/ecs-task-*.json`). CI/CD **does not** read those files; it copies the **live** task definition from AWS and only changes the image tag.

**Trouble:** Worker crash loop `invalid userinfo` on `DATABASE_URL`.  
**Fix:** URL-encode the DB password (`urllib.parse.quote(..., safe='')`), re-register task defs. After that, services ran 1/1. Later they were scaled to 0 on purpose.

**Trouble:** `update-service --task-definition` must be `applyflow-api:3` (family + revision), not a glued full ARN.

---

## 8. Load balancer (created, then deleted)

Built so the API had a **stable HTTP URL** when ECS tasks (and their IPs) recycle.

| Resource | Value |
|----------|--------|
| ALB | `applyflow-api-alb` (internet-facing, application) |
| ALB SG | Security group (80 and 443 from `0.0.0.0/0`) |
| Target group | `applyflow-api-tg`, target-type **ip**, port 8080, health `/health` |
| Listener | HTTP 80 → target group |

Health check succeeded: `curl http://<alb-dns>/health` → `{"status":"ok"}`.

**Trouble:** Commands used placeholders like `sg-ALB`; AWS rejected `InvalidGroupId.Malformed`.  
**Fix:** Always paste the real security group ID.

On 13 Sep 2026 the ALB was **deleted** to stop idle ALB billing. ECS API `loadBalancers` was set to `[]` first. Recreate steps (new ALB gets a **new DNS name**) are in the last section of this file.

---

## 9. Custom domain and HTTPS (not finished)

Goal was a custom domain for the API and frontend. DNS for the domain is not in Route 53.

ACM certificate requested in **`ap-south-2`** (same region as ALB).

Validation CNAME (never applied):

| Host | Points to |
|------|-----------|
| Validation record | ACM validation endpoint |

**Trouble:** DNS provider's admin panel has no CNAME UI. Real DNS is accessed through a different interface; CNAME is under Name Servers/DNS → Modify DNS Zone. This was treated as a **blocker**. Custom domain dropped.

**Trouble:** Google OAuth **Gmail send** is a sensitive scope. Redirect URI must be `https://` except `http://localhost`. ALB HTTP URL was rejected: *Invalid Redirect: URI must use https://*.

**Attempted fix:** CloudFront in front of the ALB with the default `*.cloudfront.net` cert (`infra/cloudfront-api.json`).

**Trouble:** `CreateDistribution` → `AccessDenied: Your account must be verified before you can add new CloudFront resources.`  
**Fix in progress:** AWS support case opened. Do not retry CloudFront until Support verifies the account.

Free subdomains (DuckDNS, etc.) generally cannot add the extra ACM CNAME, so they were not used.

Until CloudFront (or any HTTPS hostname) exists, **Sign in with Google against the AWS API cannot work**. Local `http://localhost:8080/auth/google/callback` still works.

---

## 10. CI/CD (GitHub Actions)

Added:

- `.github/workflows/deploy.yml`
- `infra/github-oidc-trust.json`
- `infra/github-actions-deploy-policy.json`

Pipeline on push to `main`:

1. `go vet` / `go test`
2. Assume IAM role `applyflow-github-actions` via **OIDC** (no long-lived keys in GitHub)
3. Build API + worker images, push ECR tagged with **git SHA** and `latest`
4. `describe-task-definition` → set image → `register-task-definition` → `update-service --force-new-deployment`

IAM:

- OIDC provider: GitHub Actions OIDC provider
- Role: `applyflow-github-actions`

**Trouble:** `Could not assume role with OIDC: Not authorized to perform sts:AssumeRoleWithWebIdentity`.

**Fixes:**

1. Trust policy must allow **`sts:TagSession`** as well as `AssumeRoleWithWebIdentity` (the AWS credentials action tags the session).
2. Deploy job needs `permissions: id-token: write`.
3. Repos created after **15 July 2026** use immutable OIDC `sub` claims:  
   `repo:murali-vivek@<owner-id>/applyFlow@<repo-id>:ref:refs/heads/main`  
   The old pattern `repo:murali-vivek/applyFlow:*` does not match. Trust policy now allows both that and `repo:murali-vivek@*/applyFlow@*:*`.

After that, a run built images, pushed ECR, and registered new task definition revisions.

**Trouble:** Services had `desiredCount: 0`, so deploy updated task defs but started **no tasks**. That was intentional (cost). The workflow must **not** force `--desired-count 1` or every push would start Fargate.

---

## 11. Cost-related shutdown (13 Sep 2026)

1. ECS API + worker left at **desired 0**.
2. ALB detached from the API service, then **deleted**.
3. RDS **kept**.
4. Bastion already **stopped**.

Turn ECS back on later:

```bash
export AWS_PROFILE=muraliVivek AWS_REGION=ap-south-2
aws ecs update-service --cluster applyflow --service applyflow-api --desired-count 1 --force-new-deployment
aws ecs update-service --cluster applyflow --service applyflow-worker --desired-count 1 --force-new-deployment
```

Then recreate the ALB (below) before expecting a public URL.

---

## Resource cheat sheet

| Kind | Name / ID |
|------|-----------|
| Region | `ap-south-2` |
| VPC | Default VPC |
| Subnets | Public subnets in default VPC |
| S3 | `applyflow-uploads` |
| SQS | `applyflow-email-queue` |
| RDS | `applyflow-db` |
| RDS SG | Private security group |
| Bastion | `applyflow-bastion` (stopped) |
| Bastion SG | Bastion security group |
| ECS cluster | `applyflow` |
| ECS SG | ECS tasks security group |
| ALB SG (kept) | ALB security group |
| Target group | `applyflow-api-tg` |
| Log group | `/ecs/applyflow` |
| RDS Enhanced Monitoring logs | `RDSOSMetrics` (RDS host metrics, not app logs) |
| GitHub deploy role | `applyflow-github-actions` |
| ACM (pending DNS) | Certificate in ap-south-2 |
| CloudFront support | AWS support case opened |

---

## Recreate the load balancer

```bash
export AWS_PROFILE=<your-profile> AWS_REGION=ap-south-2

aws elbv2 create-load-balancer \
  --name applyflow-api-alb \
  --type application \
  --scheme internet-facing \
  --subnets <subnet-ids> \
  --security-groups <alb-security-group-id>
```

Wait until `State` is `active`. Copy `Arn` and `DNSName`. Then:

```bash
aws elbv2 create-listener \
  --load-balancer-arn PASTE_ALB_ARN \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=<target-group-arn>

aws ecs update-service \
  --cluster applyflow \
  --service applyflow-api \
  --load-balancers targetGroupArn=<target-group-arn>,containerName=api,containerPort=8080 \
  --health-check-grace-period-seconds 60
```

`infra/cloudfront-api.json` still points at the **old** ALB DNS. Update that origin before creating CloudFront.

---

## Still to do

1. Support verifies CloudFront → `create-distribution` → `https://dxxxx.cloudfront.net` → Google redirect URI + `GOOGLE_REDIRECT_URL` in ECS.
2. Recreate ALB, scale ECS to 1, confirm `/health`.
3. Deploy React frontend (S3; CloudFront for the UI when allowed).
4. Optional: Secrets Manager instead of env in task definitions; restrict ECS 8080 to the ALB SG only; CloudWatch alarms.

---

## Interview one-liner

Personal app on **ECS Fargate** (API + SQS worker), **private RDS** via bastion, **S3/SQS**, **ALB** for a stable URL, **GitHub Actions OIDC** to ECR/ECS. Hit Google’s **HTTPS** requirement for Gmail scopes and an **unverified CloudFront** account; parked ALB/Fargate to stay inside the free trial until the UI and HTTPS are ready.
