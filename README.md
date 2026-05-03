# local-harness

Servidor MCP em Go inspirado no [`anyharness`](https://github.com/vinilana/anyharness) para expor o harness de desenvolvimento como protocolo unico e agnostico ao cliente de IA.

## Objetivo

Consolidar guides, sensors, judges, contracts, sessions e steering loop em uma unica surface MCP, acessivel por Cursor, OpenCode, Claude Code, Codex ou qualquer agente compativel com MCP.

## Stack

- Go 1.26
- Transporte stdio (exclusivo no MVP)
- Sem chamadas a LLM no servidor

## Estrutura

```
cmd/mcp/              # entrypoint
internal/
  mcp/                # protocolo JSON-RPC + MCP
  harness/fs/         # abstracao de .harness/ + fsnotify watcher
  guides/             # resources de guides (rules/, skills/, subagents/)
  sensors/            # tools sensor.* + adapters built-in
  judges/             # tools judge.* (review + record)
  contracts/          # tools contract.* (specs + tasks)
  sessions/           # tools session.*
  steering/           # tool harness.steer.suggest
  workflows/          # prompts MCP (workflow.{id})
  common/             # tipos e erros compartilhados
.harness/             # harness de exemplo
```

## Instalacao

```bash
go install github.com/ronaldofjc/local-harness/cmd/mcp@latest
```

Ou clone e build local:

```bash
git clone https://github.com/ronaldofjc/local-harness.git
cd local-harness
go build -o mcp ./cmd/mcp
```

## Configuracao

### Cursor

Crie ou edite `.cursor/mcp.json` na raiz do projeto:

```json
{
  "mcpServers": {
    "go-harness": {
      "command": "mcp",
      "env": {
        "HARNESS_ROOT": ".harness"
      }
    }
  }
}
```

### OpenCode

Adicione ao `opencode.json`:

```json
{
  "mcp_servers": [
    {
      "name": "go-harness",
      "command": "mcp",
      "env": {
        "HARNESS_ROOT": ".harness"
      }
    }
  ]
}
```

## Uso

1. Crie a pasta `.harness/` na raiz do projeto (veja o exemplo em `.harness/` deste repo).
2. Registre o servidor no seu cliente MCP.
3. Use as tools MCP para descobrir e executar o harness.

## Tools MCP Disponiveis

### Sensors

- `sensor.list` — lista sensores com filtros por `kind` e `regulation`
- `sensor.run` — executa um sensor pelo ID
- `sensor.register` — adiciona/atualiza um sensor

### Judges

- `judge.list` — lista rubrics disponiveis
- `judge.review` — renderiza prompt+schema+contexto (fase 1, sem LLM no servidor)
- `judge.record` — recebe verdict do cliente, valida pelo schema, retorna envelope normalizado

### Contracts

- `contract.spec.validate` — orquestra checks (sensors inline + judges pendentes)
- `contract.task.next` — retorna proxima task pendente da spec
- `contract.task.complete` — marca task como completed com evidencias

### Sessions

- `session.start` — inicia uma nova sessao append-only
- `session.append` — adiciona um evento a uma sessao
- `session.get` — le header e eventos de uma sessao

### Steering

- `harness.steer.suggest` — analisa o steering log e sugere novos guides

### Resources

- `harness://guides/{kind}/{id}` — guides (kind: rules, skills, subagents)
- `harness://workflows/{id}` — workflows (fallback)

### Prompts

Os workflows em `.harness/workflows/*.md` sao expostos como prompts MCP com prefixo `workflow.`:
- `workflow.PREVC` — Plan, Research, Execute, Verify, Commit
- `workflow.bug-fix` — investigacao → reproduzir → fix → spec/regression → completar

Tambem disponiveis como resources em `harness://workflows/{id}`.

## Sensores Built-in

| ID | Comando | Adapter | Regulacao |
|----|---------|---------|-----------|
| `go-test` | `go test -json ./...` | go-test | maintainability |
| `staticcheck` | `staticcheck ./...` | staticcheck | maintainability |
| `govet` | `go vet ./...` | govet | maintainability |
| `gofmt` | `gofmt -l .` | gofmt | maintainability |
| `go-bench` | `go test -bench=. -benchmem ./...` | go-bench | performance |
| `dep-cruiser` | comando custom com ARCH_CHECK | dep-cruiser | architecture |
| `task-harness` | `task --taskfile harness/Taskfile.yml test` | task-harness | fitness |

### Adapters Disponiveis

- `go-test` — parseia `go test -json` (pass/fail/skip)
- `staticcheck` — parseia output do staticcheck (SAxxxx)
- `govet` — parseia output do `go vet`
- `gofmt` — detecta arquivos nao formatados
- `go-bench` — parseia benchmarks Go
- `dep-cruiser` — detecta violacoes arquiteturais (handler→repository)
- `task-harness` — executa tasks do Taskfile
- `passthrough` — repassa output cru sem normalizacao

## Workflows Disponiveis

Workflows em `.harness/workflows/*.md` sao expostos como prompts MCP com prefixo `workflow.`:

```
workflow.PREVC     — Plan, Research, Execute, Verify, Commit
workflow.bug-fix   — investigacao → reproduzir → fix → spec/regression → completar
```

Tambem disponiveis como resources em `harness://workflows/{id}`.

```yaml
id: example-feature
title: Feature example
acceptanceCriteria:
  - O codigo deve compilar
  - Os testes devem passar
checks:
  - kind: sensor
    id: go-test
  - kind: judge
    rubric: spec-adherence
tasks:
  - id: implement-handler
    description: Implementar handler
    acceptanceRefs: [0]
```

## Exemplo de Session Event

Eventos em `.harness/.local/sessions/*.jsonl`:

```json
{"type":"tool_call","tool":"sensor.run","input":{"id":"gofmt","target":"."},"passed":true}
{"type":"sensor_run","sensor":"gofmt","passed":true}
{"type":"judge_review","rubric":"spec-adherence","verdict":"pass"}
{"type":"decision","action":"commit","reason":"All checks passed"}
{"type":"human_intervention","note":"Rebase required before merge"}
```

## Fluxo End-to-End

```
# 1. Descobrir sensores
sensor.list

# 2. Rodar sensor
sensor.run({ id: "gofmt", target: "." })

# 3. Preparar judge review
judge.review({ rubric_id: "spec-adherence", target: "internal/foo" })

# 4. Iniciar sessao
session.start({ workflow: "PREVC", contract_id: "example-feature" })

# 5. Pegar proxima task
contract.task.next({ spec_id: "example-feature" })

# 6. Completar task com evidencias
contract.task.complete({
  task_id: "implement-handler",
  evidence: [
    { kind: "sensor_run", sensor: "gofmt", passed: true },
    { kind: "note", text: "Review concluida" }
  ]
})

# 7. Sugerir melhorias no harness
harness.steer.suggest({ windowDays: 7 })
```

## Testes

```bash
go test ./...
```

## Roadmap

- [x] Fase 0: Protocolo MCP basico
- [x] Fase 1: Guides e Sensors
- [x] Fase 2: Judges e Contracts
- [x] Fase 3: Sessions, Steering e Workflows
- [x] Fase 4: Integracao e Dogfooding

## Licenca

MIT
