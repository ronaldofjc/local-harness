# AGENTS.md — local-harness

## Sobre este projeto

Servidor MCP em Go que expoe o harness de desenvolvimento (guides, sensors, judges, contracts, sessions) como protocolo unico agnostico ao cliente de IA.

Inspirado no [`anyharness`](https://github.com/vinilana/anyharness) (TypeScript), reescrito em Go para alinhar com a stack do workspace.

## Estado atual de implementacao

- [x] Fase 0: Protocolo MCP basico (stdio, initialize, tools/list, resources/list)
- [x] Fase 1: Guides e Sensors (fsnotify watcher, 5 adapters built-in, tools sensor.*)
- [x] Fase 2: Judges e Contracts (JSON Schema validator, rubric loader, spec/task orchestration)
- [x] Fase 3: Sessions, Steering Loop e Workflows
- [x] Fase 4: Integracao e Dogfooding

## O que foi entregue na Fase 3

1. **Sessions** (`internal/sessions/`)
   - `session.start` — cria sessao `.jsonl` em `.harness/.local/sessions/`
   - `session.append` — adiciona evento a sessao
   - `session.get` — le header + eventos
   - Schemas de eventos: `tool_call`, `sensor_run`, `judge_review`, `decision`, `human_intervention`

2. **Steering Loop** (`internal/steering/`)
   - `steering/log.go` — append-only log em `.harness/.local/steering/log.jsonl`
   - `steering/suggest.go` — heuristica de agregacao de violations e sugestao de novos guides
   - Tool MCP: `harness.steer.suggest`
   - Integracao automatica: `sensor.run` e `judge.record` logam no steering log

3. **Workflows** (`internal/workflows/`)
   - Loader de `.harness/workflows/*.md`
   - Exposicao como `prompts/` MCP (`workflow.PREVC`, `workflow.bug-fix`)
   - Fallback como resource `harness://workflows/{id}`
   - Suporte a `prompts/get`

## O que foi entregue na Fase 4

1. **Configuracao no workspace**
   - `.cursor/mcp.json` configurado para apontar para o binario do projeto
   - `opencode.json` atualizado com `mcp_servers`

2. **Dogfooding**
   - Teste end-to-end integrado (`TestEndToEnd_Flow`): sensor.run -> judge.review -> contract.task.next -> contract.task.complete
   - `.harness/` de exemplo completo com guides, sensors, judges, contracts, workflows

3. **Documentacao**
   - README atualizado com todas as tools, fluxo end-to-end e exemplos
   - Tags JSON adicionadas as structs expostas via MCP para garantir serializacao correta

## Comandos uteis

```bash
# Build
go build -o mcp ./cmd/mcp

# Testes
go test ./...

# Executar servidor (stdio)
./mcp
```

## Estrutura de pastas

```
cmd/mcp/              # entrypoint
internal/
  mcp/                # protocolo JSON-RPC + MCP
  harness/fs/         # watcher fsnotify
  guides/             # resources guides
  sensors/            # tools sensor.* + adapters
  judges/             # tools judge.*
  contracts/          # tools contract.*
  sessions/           # (Fase 3) tools session.*
  steering/           # (Fase 3) harness.steer.suggest
  workflows/          # (Fase 3) prompts MCP
  common/             # tipos compartilhados
.harness/             # harness de exemplo
```

## Decisoes arquiteturais

- Transporte stdio exclusivo (sem HTTP/SSE)
- Sem chamadas a LLM no servidor (inferencia no cliente MCP)
- File-system first (sem banco no MVP)
- Go 1.26

## Dependencias

```
github.com/fsnotify/fsnotify v1.10.0
github.com/xeipuuv/gojsonschema v1.2.0
gopkg.in/yaml.v3 v3.0.1
```

## Plano completo

Veja `../../vault/20-projects/local-harness-plan.md`

## Wiki

Veja `../../vault/wiki/log.md` para historico de execucao.
