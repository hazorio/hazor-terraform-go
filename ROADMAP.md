# Terraform Provider Hazor — Roadmap

## Estado Actual
- **25 resources** implementados
- **0 data sources**
- Framework: terraform-plugin-framework (Go)
- Registro: pendiente de publicar en registry.terraform.io

---

## Fase 1 — Core Completeness (recursos que bloquean production workflows)

**Objetivo:** Cualquier infraestructura deployable desde la UI se puede hacer desde Terraform.

### Resources (6)

| Resource | Descripcion | API Endpoint | Dependencia |
|---|---|---|---|
| `hazor_target_group` | Target group para load balancers | `POST/GET/PUT/DELETE /target-groups` | `hazor_load_balancer` |
| `hazor_launch_template` | Templates para instancias | `POST/GET/PUT/DELETE /templates` | `hazor_auto_scaling_group` |
| `hazor_auto_scaling_group` | Auto scaling groups | `POST/GET/PUT/DELETE /auto-scaling` | `hazor_launch_template` |
| `hazor_image` | Custom machine images | `POST/GET/DELETE /images` | `hazor_instance` |
| `hazor_secret` | Secrets Manager | `POST/GET/PUT/DELETE /secrets` | — |
| `hazor_api_key` | API keys para automation | `POST/GET/DELETE /api-keys` | — |

### Data Sources (10)

| Data Source | Descripcion |
|---|---|
| `data.hazor_vpc` | Lookup VPC por ID o nombre |
| `data.hazor_subnet` | Lookup subnet por ID o filtros |
| `data.hazor_image` | Lookup image por nombre/familia |
| `data.hazor_instance` | Lookup instancia por ID o nombre |
| `data.hazor_security_group` | Lookup SG por nombre o VPC |
| `data.hazor_key_pair` | Lookup key pair por nombre |
| `data.hazor_availability_zones` | Listar AZs disponibles |
| `data.hazor_instance_types` | Listar instance types |
| `data.hazor_dns_zone` | Lookup zona DNS |
| `data.hazor_load_balancer` | Lookup LB por ID o nombre |

### Acceptance Tests
- Tests para cada resource nuevo (create, read, update, delete, import)
- Tests para cada data source (basic lookup)

**Entregable:** El usuario puede hacer un deploy completo con auto-scaling desde Terraform.

---

## Fase 2 — Database & Services (recursos de servicios gestionados)

**Objetivo:** Todos los servicios gestionados tienen resource Terraform.

### Resources (4)

| Resource | Descripcion | API Endpoint |
|---|---|---|
| `hazor_mongodb` | Managed MongoDB clusters | `POST/GET/PUT/DELETE /mongodb` |
| `hazor_docker_cluster` | Docker Swarm clusters | `POST/GET/PUT/DELETE /docker` |
| `hazor_mq_queue` | Message queues | `POST/GET/PUT/DELETE /mq` |
| `hazor_global_load_balancer` | Global LB multi-region | `POST/GET/PUT/DELETE /global-lb` |

### Data Sources (6)

| Data Source | Descripcion |
|---|---|
| `data.hazor_database` | Lookup DB cluster |
| `data.hazor_redis` | Lookup Redis instance |
| `data.hazor_supabase` | Lookup Supabase instance |
| `data.hazor_k8s_cluster` | Lookup K8s cluster |
| `data.hazor_postgresml` | Lookup PostgresML |
| `data.hazor_volume` | Lookup volume por nombre |

**Entregable:** Stack completo de DB + cache + messaging desde Terraform.

---

## Fase 3 — Security & IAM (RBAC, VPN, compliance)

**Objetivo:** Governance y seguridad como código.

### Resources (5)

| Resource | Descripcion | API Endpoint |
|---|---|---|
| `hazor_iam_policy` | IAM policies (RBAC) | `POST/GET/PUT/DELETE /iam` |
| `hazor_iam_role` | IAM roles | `POST/GET/PUT/DELETE /iam/roles` |
| `hazor_vpn_gateway` | Site-to-site VPN | `POST/GET/PUT/DELETE /vpn` |
| `hazor_spot_request` | Spot instance requests | `POST/GET/DELETE /spot-requests` |
| `hazor_reserved_instance` | Reserved capacity | `POST/GET/DELETE /reserved-instances` |

### Data Sources (3)

| Data Source | Descripcion |
|---|---|
| `data.hazor_iam_policy` | Lookup policy por nombre |
| `data.hazor_secret` | Lookup secret (value redacted) |
| `data.hazor_elastic_ip` | Lookup EIP disponible |

**Entregable:** Full RBAC + VPN + cost optimization desde Terraform.

---

## Fase 4 — Advanced Networking (cuando se habiliten en la UI)

**Objetivo:** Networking avanzado como código (actualmente oculto en UI).

### Resources (8) — implementar cuando se habiliten

| Resource | Descripcion | API Endpoint |
|---|---|---|
| `hazor_firewall_policy` | Distributed Firewall (DFW) | `/dfw` |
| `hazor_route_table` | Custom routing | `/route-tables` |
| `hazor_placement_group` | VM placement constraints | `/placement-groups` |
| `hazor_overlay_tunnel` | VXLAN/Geneve tunnels | `/overlay` |
| `hazor_routing_peer` | BGP/OSPF peers | `/routing` |
| `hazor_edge_gateway` | Edge routing appliance | `/edge-gateway` |
| `hazor_service_chain` | Network function chaining | `/service-chains` |
| `hazor_federation_site` | Multi-site federation | `/federation` |

**Entregable:** Full NSX-style networking desde Terraform.

---

## Fase 5 — Polish & Registry

**Objetivo:** Publicar en Terraform Registry y documentación completa.

### Tareas

- [ ] Import support para todos los resources (`terraform import`)
- [ ] Documentación completa en formato registry (examples, argument reference)
- [ ] Generar docs con `tfplugindocs`
- [ ] CI/CD pipeline (GoReleaser + GitHub Actions)
- [ ] Firmar con GPG key para registry
- [ ] Publicar en registry.terraform.io como `hazorio/hazor`
- [ ] Example modules:
  - `modules/web-app` — VPC + subnet + instance + LB + DNS
  - `modules/database-stack` — DB + Redis + Supabase
  - `modules/k8s-platform` — K8s + registry + runners

---

## Resumen de Fases

| Fase | Resources | Data Sources | Total Items | Prioridad |
|---|---|---|---|---|
| 1 — Core | 6 | 10 | 16 | Inmediata |
| 2 — Services | 4 | 6 | 10 | Alta |
| 3 — Security | 5 | 3 | 8 | Media |
| 4 — Networking | 8 | 0 | 8 | Cuando se habilite |
| 5 — Polish | 0 | 0 | CI/CD + Registry | Después de Fase 2 |
| **Total** | **48** | **19** | **42 + polish** | |

## Cobertura Final Proyectada
- Resources: **48** (de 25 actuales → 92% coverage)
- Data Sources: **19** (de 0 actuales)
- Import: todos los resources
