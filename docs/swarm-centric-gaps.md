# Swarm-Centric Gaps To Track

Docksmith and Permanu can build higher-level production behavior around Docker
Swarm, but some Kubernetes-style capabilities are not native Swarm features.
These should be treated as platform gaps to compensate for outside Swarm, not
as defects to patch inside Swarm itself.

## Native Swarm Limits

- No StatefulSet-style stable ordinal identities such as `postgres-0`.
- No native operator or CRD ecosystem for databases, queues, and storage.
- No built-in Horizontal Pod Autoscaler equivalent.
- No CSI ecosystem parity for portable storage provisioning.
- No PodDisruptionBudget equivalent.
- No admission-controller ecosystem for policy enforcement.
- Weaker topology and scheduling primitives than Kubernetes.
- No generic stateful failover abstraction.
- No native app-aware autoscaling controller.
- Secret rotation semantics require platform-managed rollout policy.

## Required Platform Compensation

- Docksmith blueprints must encode stateful operational knowledge explicitly.
- Permanu must reject unsafe production specs before they reach the agent.
- Autoscaling must live in Permanu, gated by quota, billing, and approval.
- Dwaar should provide traffic metrics and health-aware routing, not scaling
  authority.
- Stateful HA must be blueprint-specific, for example Postgres via Patroni or
  pg_auto_failover, Redis via Sentinel or Cluster, and RabbitMQ via a known
  cluster mode.
- Production stateful blueprints must require backup, restore, and verification
  behavior.
- Production storage must require explicit placement constraints or an external
  volume driver.
- Deployment updates must use blueprint-specific rollout rules instead of
  generic stateless rolling updates.

## Deferred Work

- Swarm executor in the agent.
- Permanu autoscaling controller.
- Dwaar healthy-upstream pool integration for replicated Swarm services.
- Storage driver compatibility matrix.
- Stateful blueprint HA catalog.
- Runtime secret rotation and restart policy.
- Production admission/policy layer over compiled Docksmith plans.
