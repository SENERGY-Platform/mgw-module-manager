mgw-module-manager
=======

mgw-module-manager is the central service of the MGW ("multi gateway") edge-gateway stack.
It allows the installation of **modules** — containerized add-on applications.

State is kept in a MySQL database. `lib/` is a separate Go module containing the API
models and HTTP clients, so other services, apps and modules can interact with the service.

## Table of contents

- [Service interactions](#service-interactions)
- [Concepts](#concepts)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [HTTP API](#http-api)

## Service interactions

| Service | Used for |
| --- | --- |
| [container-engine-wrapper](https://github.com/SENERGY-Platform/mgw-container-engine-wrapper) (CEW) | images, containers, volumes |
| [host-manager](https://github.com/SENERGY-Platform/mgw-host-manager) (HM) | host resources (applications, serial devices) |
| [secret-manager](https://github.com/SENERGY-Platform/mgw-secret-manager) (SM) | secret values and secret mounts |
| [core-manager](https://github.com/SENERGY-Platform/mgw-core-manager) (CM) | registering module HTTP endpoints with the gateway's reverse proxy |

```mermaid
flowchart LR
    UI[UI / Apps] -->|API| MM
    MOD[modules] -->|restricted API| MM
    MM[module-manager]
    MM --> CEW[container-engine-wrapper]
    MM --> HM[host-manager]
    MM --> SM[secret-manager]
    MM --> CM[core-manager]
    MM --> DB[(Database)]
    REPO[repository handlers] -.->|modules| MM
```

## Concepts

### Repositories

Sources of modules, each with a source name, priority and one or more *channels*. Two backends
are implemented:

- **GitHub**: repositories containing modules fetched from GitHub; channels map to directories in the repositry. (E.g. https://github.com/SENERGY-Platform/mgw-module-repository)
- **host directory**: a local directory, source `localhost`, channel `default`, for side-loading
  and development.

### Modules

A Modfile declares everything a module needs: services (containers), auxiliary services,
dependencies on other modules, service references (env var injection
with templates), typed configs with user-input metadata, files and file groups, volumes, secrets,
host resources, ports and HTTP endpoints.

### Change requests

Installs, updates and removals go through a deliberate two-phase flow:

1. Create a change request — dependencies are resolved and module variants selected from the
   repositories.
2. Review the resulting install / change / remove plan, then execute it as a job, or cancel it.

Convenience endpoints exist for updating everything and for counting available updates.

### Deployment

Deploying a module takes user input (configs, global configs, secrets, host
resources, files) and then:

- generates a deployment id and container dns aliases,
- creates volumes,
- materializes config files and file groups on disk as bind mounts,
- mounts secrets via SM,
- creates the containers via CEW, injecting the container dns aliases of resolved dependencies,
- registers the module's HTTP endpoints with CM.
- ...

### Auxiliary deployments

Extra containers a *running* module can spawn, with their own labels,
configs and volumes. Images are restricted to the `auxImageSources` declared in the module's
Modfile.

### Advertisements

Key/value records a deployment publishes under a reference and other deployments or external apps can query — a
lightweight discovery mechanism.

### Global configs

Named, typed values shared by any number of deployments.

### Jobs

Every long-running operation (module change, deployment create / update / delete, repository
refresh, auxiliary deployment operations) runs asynchronously and returns a job. Job results are
cached and removed once they exceed the configured maximum age.

### Runtime monitors

Two handlers — one for deployments, one for auxiliary deployments — compare the desired
state in the database with the actual container state reported by CEW and start or
stop containers accordingly.

## Configuration

### Service

| Env var | Type | Default | Description |
|---|---|---|---|
| `SERVER_PORT` | uint | `80` | HTTP port the API server listens on. |
| `MANAGER_ID_PATH` | string | `./service/mid` | File path where the manager's own ID is stored/read. |
| `CORE_ID` | string | – | ID of the MGW core this manager belongs to. |
| `MODULE_CONTAINER_NETWORK` | string | – | Container network that module containers are attached to. |
| `USE_UTC` | bool | `true` | Use UTC for timestamps. |
| `JOB_POLL_INTERVAL` | string | `500ms` | Interval for polling job state. |
| `IMAGE_NAME_ESCAPE_DEPTH` | int | `1` | Path-segment depth used when escaping image names. |
| `HOST_DEPLOYMENTS_PATH` | string | – | Path on the host to the deployments directory (for bind mounts). |
| `HOST_SECRETS_PATH` | string | – | Path on the host to the secrets directory (for bind mounts). |

### Logger

| Env var | Type | Default                                               | Description |
|---|---|-------------------------------------------------------|---|
| `HTTP_ACCESS_LOG` | bool | `false`                                               | Enable HTTP access logging. |
| `LOGGER_HANDLER` | string | `text`                                                | Log handler/format (`text`, `colored-text`, …). |
| `LOGGER_LEVEL` | string | `info`                                                | Log level. |
| `LOGGER_TIME_FORMAT` | string | `RFC3339Nano` (`2006-01-02T15:04:05.999999999Z07:00`) | Timestamp layout. |
| `LOGGER_TIME_UTC` | bool | `true`                                                | Emit log timestamps in UTC. |
| `LOGGER_FILE_PATH` | string | –                                                     | Write logs to this file instead of stdout. |
| `LOGGER_ADD_SOURCE` | bool | -                                                     | Include source file/line in records. |
| `LOGGER_ADD_META` | bool | -                                                     | Include metadata attributes in records. |
| `LOGGER_TRIM_FORMAT` | string | –                                                     | Trim spec (e.g. `20:[...]:10`), applied to the log message. |
| `LOGGER_TRIM_ATTRIBUTES` | string | –                                                     | Comma-separated attribute keys that should also be trimmed. |

### MGW core

| Env var | Type | Default | Description |
|---|---|---|---|
| `MGW_CEW_BASE_URL` | string | – | Base URL of the container engine wrapper. |
| `MGW_CM_BASE_URL` | string | – | Base URL of the core manager. |
| `MGW_HM_BASE_URL` | string | – | Base URL of the host manager. |
| `MGW_SM_BASE_URL` | string | – | Base URL of the secret manager. |
| `MGW_HTTP_TIMEOUT` | duration (string) | `30s` | HTTP timeout for calls to the core services. |

### Database

| Env var | Type | Default | Description                          |
|---|---|---|--------------------------------------|
| `DATABASE_ADDRESS` | string | – | Database host/address.               |
| `DATABASE_NAME` | string | `module_manager` | Database name.                       |
| `DATABASE_USER` | string | – | Database user.                       |
| `DATABASE_PASSWORD` | secret (string) | – | Database password.                   |
| `DATABASE_TIMEOUT` | duration (string) | `30s` | Query/connection timeout.            |
| `DATABASE_MAX_OPEN_CONNECTIONS` | int | `25` | Max open connections in the pool.    |
| `DATABASE_MAX_IDLE_CONNECTIONS` | int | `25` | Max idle connections in the pool.    |
| `DATABASE_CONNECTION_MAX_LIFETIME` | duration (string) | `5m` | Max lifetime of a pooled connection. |

### Modules handler

| Env var | Type | Default | Description |
|---|---|---|---|
| `MODULES_HANDLER_WORKDIR_PATH` | string | `./modules` | Working directory for module data. |

### Deployments handler

| Env var | Type | Default         | Description |
|---|---|-----------------|---|
| `DEPLOYMENTS_HANDLER_WORKDIR_PATH` | string | `./deployments` | Working directory for deployment data. |
| `DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_STARTUP_DELAY` | duration (string) | -               | Delay before the runtime monitor starts. |
| `DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_LOOP_DELAY` | duration (string) | `5s`            | Delay between runtime monitor iterations. |

### Aux deployments handler

| Env var | Type | Default | Description |
|---|---|---------|---|
| `AUX_DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_STARTUP_DELAY` | duration (string) | -       | Delay before the aux runtime monitor starts. |
| `AUX_DEPLOYMENTS_HANDLER_RUNTIME_MONITOR_LOOP_DELAY` | duration (string) | `5s`    | Delay between aux runtime monitor iterations. |

### Host dir repository handler

| Env var | Type | Default | Description |
|---|---|---|---|
| `HOST_DIR_HANDLER_WORKDIR_PATH` | string | `./repositories/host_dir` | Working directory for the host-dir repository. |
| `HOST_DIR_HANDLER_PRIORITY` | int | `0` | Priority of this repository relative to others. |

### GitHub repositories handler

| Env var | Type              | Default | Description |
|---|-------------------|---|---|
| `GITHUB_HANDLER_BASE_URL` | string            | `https://api.github.com` | GitHub API base URL. |
| `GITHUB_HANDLER_WORKDIR_PATH` | string            | `./repositories/github` | Working directory for GitHub repository data. |
| `GITHUB_HANDLER_HTTP_TIMEOUT` | duration (string) | `1m` | HTTP timeout for GitHub API calls. |

### Jobs handler

| Env var | Type | Default | Description |
|---|---|---|---|
| `JOBS_HANDLER_MAX_JOB_AGE` | duration (string) | `24h` | Age after which finished jobs are removed. |
| `JOBS_HANDLER_CLEANUP_LOOP_DELAY` | duration (string) | `5m` | Delay between job cleanup runs. |

## HTTP API

| Set | Mounted at | Audience                                                                                                                          |
| --- | --- |-----------------------------------------------------------------------------------------------------------------------------------|
| standard | `/` | management surface for the UI and external apps: modules, repositories, deployments, global configs, change requests, job results |
| restricted | `/restricted` | calls coming from module containers: auxiliary deployments and advertisements                                                     |
| shared | both | read access to auxiliary deployments and advertisements, jobs, health, service info                                               |

Every response carries `X-Core-Id`, `X-Manager-Id`, `X-Runtime-Id`, `X-Version` and `X-Service`
headers; `X-Request-Id` is honoured or generated and threaded through the structured logs together
with the runtime and job ids.

### Creating and Executing a Modules Change Request

HTTP interaction between a client and the service API. The pending change request is a singleton on the
service: `POST` creates or replaces it, `PATCH` executes and clears it, `DELETE` discards it.

The request type is determined by the `ChangeRequestItem` fields in the `POST` body:

| Intent       | Request item                                  | Preview list populated |
|--------------|-----------------------------------------------|------------------------|
| install/change | `{"id": ..., "source": ..., "channel": ...}`  | `install`/`change`             |
| update       | `{"id": ..., "update": true}`                 | `change`               |
| delete       | `{"id": ..., "remove": true}`                 | `remove`               |

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant A as module-manager

    Note over C,A: 1. Create change request
    alt install / change / update / delete
        C->>A: m=POST, p="/modules-change-request", b=[]ChangeRequestItem
    else update all installed modules
        C->>A: m=POST, p="/modules-change-request?update_all=true"
    end
    alt error
        A-->>C: s=503(active job)|500(general error)|400(invalid or duplicate request items), b="error message"
    else accepted
        A-->>C: s=200, b=ModulesChangeRequest
    end

    Note over C,A: 2. Review pending change request (optional)
    C->>A: m=GET, p="/modules-change-request"
    alt none pending
        A-->>C: s=404, b="error message"
    else
        A-->>C: s=200, b=ModulesChangeRequest
    end

    Note over C,A: 3. Discard instead of executing (optional)
    C->>A: m=DELETE, p="/modules-change-request"
    alt none pending
        A-->>C: s=404, b="error message"
    else
        A-->>C: s=200
    end

    Note over C,A: 4. Execute
    C->>A: m=PATCH, p="/modules-change-request"
    alt error
        A-->>C: s=503(active job)|500(general error)|404(none pending), b="error message"
    else started
        A-->>C: s=200, b=Job
    end

    Note over C,A: 5. Await job completion
    loop until field "end" has non zero timestamp
        C->>A: m=GET, p="/jobs/:JOB_ID"
        A-->>C: s=200, b=Job
    end
    C->>A: m=GET, p="/jobs/:JOB_ID"
    A-->>C: s=200, b=Job

    Note over C,A: Abort the running job (optional)
    C->>A: m=PATCH, p="/jobs/:JOB_ID"
    A-->>C: s=200

    Note over C,A: 6. Fetch change report
    C->>A: m=GET, p="/results/modules-change/:JOB_ID"
    A-->>C: s=200, b=ModulesChangeJobResult
```

Per-module failures are reported in `failed` — the job itself still completes successfully. A module is
only removed while it is not deployed, so its deployment must be deleted before a `remove` item can succeed.

#### The `ChangeRequestItem`

The body of `POST /modules-change-request` is a JSON array of `ChangeRequestItem`. One item describes the
intent for exactly one module; the service turns the whole array into a single change request preview.

```json
{
  "id": "github.com/acme/mod-a",
  "source": "github.com/acme/repository",
  "channel": "main",
  "remove": false,
  "update": false
}
```

| Field | Type | Meaning |
|-------|------|---------|
| `id` | string | Module ID the item refers to. Used to look the module up in the repositories (install) or among the installed modules (update, remove). |
| `source` | string | Repository source the module is taken from. Only evaluated when neither `update` nor `remove` is set. |
| `channel` | string | Repository channel the module is taken from. Only evaluated when neither `update` nor `remove` is set. |
| `remove` | bool | Uninstall the module. `source` and `channel` are ignored. |
| `update` | bool | Update the module in place. `source` and `channel` are taken from the installed module and any values sent are ignored. |

`remove` and `update` are the two mode flags; leaving both unset selects the third mode, install/change, which is the only mode that reads `source` and `channel`.

#### How an item is processed

1. **Validation**: rejects illegal combinations and conflicting duplicates.
   The whole request fails; nothing is stored.
2. **Repository selection**: `remove` items are skipped entirely. Dependencies of
   every selected module are resolved and added to the request automatically, first from the
   highest-priority repository and channel, then from the origin repository and channel of the
   selecting module.
3. **Classification for preview**: each selected module becomes an `install` entry (not
   installed yet), a `change` entry (installed, but with a different source, channel or version), or is
   dropped (installed with the identical variant and version). `remove` entries are kept only if the module
   is actually installed and is not also selected for install or change.

An item can therefore be silently dropped — the preview is the authoritative answer to
what will actually happen. If all items are dropped, the response contains three empty lists and no
change request is stored.

### Create Deployment

HTTP interaction between a client and the service HTTP API when creating a deployment.

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant A as module-manager

    Note over C,A: 1. Get modules and dependencies

    C->>+A: m=GET, p="/deployment-request?module_ids=<csv>"
    alt error
        A-->>C: s=503(active job)|500(general error), b="error message"
    else modules
        A-->>C: s=200, b=[]Module
    end

    Note over C,A: 2. Start deployment creation

    C->>+A: m=POST, p="/deployments", b=[]DeploymentUserInput
    alt error
        A-->>C: s=503(active job)|500(general error)|400(invalid input), b="error message"
    else accepted
        A-->>C: s=200, b=Job
    end

    Note over C,A: 3. Await job completion

    loop until field "end" has non zero timestamp
        C->>A: m=GET, p="/jobs/:JOB_ID"
        A-->>C: s=200, b=Job
    end
    C->>A: m=GET, p="/jobs/:JOB_ID"
    A-->>C: s=200, b=Job

    Note over C,A: Abort the running job (optional)
    C->>A: m=PATCH, p="/jobs/:JOB_ID"
    A-->>C: s=200

    Note over C,A: 4. Fetch job result

    C->>A: m=GET, p="/results/deployments/:JOB_ID"
    A-->>C: s=200, b=DeploymentJobResult
```

### Update Deployment

HTTP interaction between a client and the service HTTP API when updating a deployment.

```mermaid
sequenceDiagram
    autonumber
    actor C as Client
    participant A as Service HTTP API

    Note over C,A: 1. Get modules
 
    alt
        C->>+A: m=GET, p="/modules?ids=<csv>"
        alt error
            A-->>C: s=503(active job)|500(general error), b="error message"
        else modules
            A-->>C: s=200, b=[]Module
        end
    else single module    
        C->>A: m=GET, p="/modules/:MOD_ID"
        alt error
            A-->>C: s=503(active job)|500(general error)|404(not found), b="error message"
        else module
            A-->>C: s=200, b=Module
        end
    end

    Note over C,A: 2. Start deployment update

    C->>+A: m=PUT, p="/deployments", b=[]DeploymentUserInput
    alt error
        A-->>C: s=503(active job)|500(general error)|400(invalid input), b="error message"
    else accepted
        A-->>C: s=200, b=Job
    end

    Note over C,A: 3. Await job completion

    loop until field "end" has non zero timestamp
        C->>A: m=GET, p="/jobs/:JOB_ID"
        A-->>C: s=200, b=Job
    end
    C->>A: m=GET, p="/jobs/:JOB_ID"
    A-->>C: s=200, b=Job

    Note over C,A: Abort the running job (optional)
    C->>A: m=PATCH, p="/jobs/:JOB_ID"
    A-->>C: s=200

    Note over C,A: 5. Fetch job result

    C->>+A: m=GET p="/results/deployments-update/:JOB_ID"
    A-->>C: s=200, b=DeploymentUpdateJobResult
```
