# local-harness

Servidor MCP em Go que expõe o harness de desenvolvimento como protocolo único e agnóstico ao cliente de IA.

## Descrição

O **local-harness** é um servidor [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) que unifica guides, sensors, judges, contracts, sessions e steering loop em uma única surface de protocolo. Ele permite que qualquer agente de IA — seja Cursor, OpenCode, Claude Code, Codex ou outro cliente MCP — descubra, execute e valide o harness de qualidade do seu projeto de forma padronizada.

O servidor opera no modelo **file-system first**: todas as definições (sensores, rubricas, specs, workflows) são arquivos no diretório `.harness/` do seu workspace, sem necessidade de banco de dados ou serviços externos.

## Objetivo

Consolidar o harness de desenvolvimento em uma surface MCP única, acessível por qualquer agente compatível, garantindo:

- **Descoberta automática** de regras, checks e workflows via filesystem
- **Execução normalizada** de sensores (testes, lint, benchmarks, validação arquitetural)
- **Avaliação estruturada** via judges com rubricas e schemas JSON
- **Rastreabilidade completa** através de sessions append-only
- **Evolução contínua** do harness via steering loop baseado em violações

## Stack

- Go 1.26
- Transporte stdio (exclusivo no MVP)
- Sem chamadas a LLM no servidor (inferência 100% no cliente MCP)
- File-system first (sem banco de dados)

## Estrutura

```
cmd/mcp/              # Entrypoint do servidor MCP (stdio)
internal/
  mcp/                # Protocolo JSON-RPC + implementação MCP
  harness/fs/         # Abstração de .harness/ + watcher fsnotify
  guides/             # Resources de guides (rules/, skills/, subagents/)
  sensors/            # Tools sensor.* + adapters built-in para normalização
  judges/             # Tools judge.* (review fase 1 + record fase 2)
  contracts/          # Tools contract.* (specs + tasks orchestration)
  sessions/           # Tools session.* (append-only jsonl)
  steering/           # Tool harness.steer.suggest (heurísticas de evolução)
  workflows/          # Prompts MCP (workflow.{id}) + loader de .md
  common/             # Tipos, erros e utilitários compartilhados
examples/
  .harness/           # Harness de exemplo completo (sensores, workflows, etc.)
```

## Instalação

### Via go install

```bash
go install github.com/ronaldofjc/local-harness/cmd/mcp@latest
```

> **Nota:** Certifique-se de que seu `$GOPATH/bin` ou `$GOBIN` esteja no `PATH`.

### Build local

```bash
git clone https://github.com/ronaldofjc/local-harness.git
cd local-harness
go build -o mcp ./cmd/mcp
```

### Verificação

```bash
./mcp --version  # ou simplesmente mcp, se instalado via go install
```

## Configuração

O `local-harness` pode ser utilizado em **qualquer ferramenta de IA compatível com MCP** (Cursor, OpenCode, Claude Code, Codex, etc.). A única configuração obrigatória é a variável de ambiente `HARNESS_ROOT`, que deve apontar para o diretório `.harness/` do seu workspace.

### Cursor

Crie ou edite `.cursor/mcp.json` na raiz do projeto:

```json
{
  "mcpServers": {
    "local-harness": {
      "command": "/caminho/para/mcp",
      "env": {
        "HARNESS_ROOT": "/caminho/do/workspace/.harness"
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
      "name": "local-harness",
      "command": "/caminho/para/mcp",
      "env": {
        "HARNESS_ROOT": "/caminho/do/workspace/.harness"
      }
    }
  ]
}
```

### Claude Code

Adicione ao arquivo de configuração do Claude Code (`~/.claude/settings.json` ou equivalente):

```json
{
  "mcpServers": {
    "local-harness": {
      "command": "/caminho/para/mcp",
      "env": {
        "HARNESS_ROOT": "/caminho/do/workspace/.harness"
      }
    }
  }
}
```

> **Dica:** Use o caminho absoluto para o binário `mcp` e para `HARNESS_ROOT` para evitar problemas de resolução de PATH.

## Uso

1. **Crie a pasta `.harness/`** na raiz do seu projeto (veja o exemplo em `examples/.harness/` deste repositório).
2. **Registre o servidor MCP** no seu cliente de IA conforme a seção de Configuração.
3. **Use as tools MCP** para descobrir e executar o harness.

### Estrutura mínima do `.harness/`

```
.harness/
├── sensors/          # Arquivos YAML com definição de sensores
├── judges/           # Arquivos YAML com rubricas de avaliação
├── contracts/        # Arquivos YAML com specs e tasks
├── workflows/        # Arquivos Markdown com workflows
└── guides/           # Arquivos Markdown com guides (rules, skills, subagents)
```

## Tools MCP Disponíveis

### Sensors

| Tool | Descrição |
|------|-----------|
| `sensor.list` | Lista sensores registrados com filtros opcionais por `kind` e `regulation` |
| `sensor.run` | Executa um sensor pelo ID e retorna output normalizado |
| `sensor.register` | Adiciona ou atualiza um sensor em `.harness/sensors/` |

### Judges

| Tool | Descrição |
|------|-----------|
| `judge.list` | Lista rubrics de judges disponíveis |
| `judge.review` | Renderiza prompt + schema + contexto para avaliação (fase 1, sem LLM no servidor) |
| `judge.record` | Recebe verdict do cliente, valida pelo schema e retorna envelope normalizado (fase 2) |

### Contracts

| Tool | Descrição |
|------|-----------|
| `contract.spec.validate` | Valida uma spec contra um artefato, orquestrando sensors e judges |
| `contract.task.next` | Retorna a próxima task pendente de uma spec |
| `contract.task.complete` | Marca uma task como completed com evidências |

### Sessions

| Tool | Descrição |
|------|-----------|
| `session.start` | Inicia uma nova sessão append-only em `.harness/.local/sessions/`. Reutiliza a sessão ativa da janela de execução, a menos que `force_new` seja `true` |
| `session.append` | Adiciona um evento a uma sessão existente |
| `session.get` | Lê o cabeçalho e todos os eventos de uma sessão |

### Steering

| Tool | Descrição |
|------|-----------|
| `harness.steer.suggest` | Analisa o steering log e sugere novos guides baseado em padrões de violations |

### Resources

- `harness://guides/{kind}/{id}` — Guides organizados por kind (ex: `rules`, `skills`, `subagents`)
- `harness://workflows/{id}` — Workflows como fallback resources

### Prompts

Os workflows em `.harness/workflows/*.md` são expostos como prompts MCP com prefixo `workflow.`:

- `workflow.PREVC` — Plan, Research, Execute, Verify, Commit
- `workflow.bug-fix` — investigação → reproduzir → fix → spec/regression → completar

## Sensores Built-in

| ID | Comando | Adapter | Regulação |
|----|---------|---------|-----------|
| `go-test` | `go test -json ./...` | `go-test` | maintainability |
| `staticcheck` | `staticcheck ./...` | `staticcheck` | maintainability |
| `govet` | `go vet ./...` | `govet` | maintainability |
| `gofmt` | `gofmt -l .` | `gofmt` | maintainability |
| `go-bench` | `go test -bench=. -benchmem ./...` | `go-bench` | performance |
| `dep-cruiser` | comando custom com `ARCH_CHECK` | `dep-cruiser` | architecture |
| `task-harness` | `task --taskfile harness/Taskfile.yml test` | `task-harness` | fitness |

### Adapters Disponíveis

| Adapter | Descrição |
|---------|-----------|
| `go-test` | Parseia `go test -json` (pass/fail/skip) |
| `staticcheck` | Parseia output do staticcheck (SAxxxx) |
| `govet` | Parseia output do `go vet` |
| `gofmt` | Detecta arquivos não formatados |
| `go-bench` | Parseia benchmarks Go |
| `dep-cruiser` | Detecta violações arquiteturais (ex: handler → repository direto) |
| `task-harness` | Executa tasks de um Taskfile |
| `passthrough` | Repassa output cru sem normalização |

## Workflows Disponíveis

Os workflows em `.harness/workflows/*.md` são expostos como prompts MCP e também como resources:

| Prompt | Descrição |
|--------|-----------|
| `workflow.PREVC` | Plan, Research, Execute, Verify, Commit |
| `workflow.bug-fix` | investigação → reproduzir → fix → spec/regression → completar |

### Exemplo de Spec (Contract)

```yaml
id: example-feature
title: Feature example
acceptanceCriteria:
  - O código deve compilar
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
# 1. Descobrir sensores disponíveis
sensor.list

# 2. Executar sensor de qualidade
sensor.run({ id: "gofmt", target: "." })

# 3. Preparar judge review
judge.review({ rubric_id: "spec-adherence", target: "internal/foo" })

# 4. Iniciar sessão rastreável
session.start({ workflow: "PREVC", contract_id: "example-feature" })

# 5. Identificar próxima task
contract.task.next({ spec_id: "example-feature" })

# 6. Completar task com evidências
contract.task.complete({
  task_id: "implement-handler",
  evidence: [
    { kind: "sensor_run", sensor: "gofmt", passed: true },
    { kind: "note", text: "Review concluída" }
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

- [x] Fase 0: Protocolo MCP básico (stdio, initialize, tools/list, resources/list)
- [x] Fase 1: Guides e Sensors (fsnotify watcher, 5+ adapters built-in, tools sensor.*)
- [x] Fase 2: Judges e Contracts (JSON Schema validator, rubric loader, spec/task orchestration)
- [x] Fase 3: Sessions, Steering Loop e Workflows
- [x] Fase 4: Integração e Dogfooding

## Licença

MIT
