# Playbook de Continuacao

## Como retomar este projeto

1. Leia `AGENTS.md` na raiz do projeto
2. Leia o plano em `../../vault/20-projects/local-harness-plan.md`
3. Verifique a fase atual no `AGENTS.md`
4. Execute `go test ./...` para garantir que tudo passa antes de comecar
5. Implemente a proxima tarefa da fase em aberto

## Fase 3 — Tarefas

### Sessions
- [ ] `internal/sessions/store.go` — append-only jsonl
- [ ] `internal/sessions/events.go` — schemas de eventos (tool_call, sensor_run, judge_review, decision, human_intervention)
- [ ] `session.start`, `session.append`, `session.get`

### Steering
- [ ] `internal/steering/log.go` — append-only log.jsonl
- [ ] `internal/steering/suggest.go` — heuristicas de sugestao
- [ ] Tool `harness.steer.suggest`

### Workflows
- [ ] `internal/workflows/loader.go` — le workflows/*.md
- [ ] `internal/workflows/prompts.go` — expoe como prompts MCP
- [ ] Workflows iniciais: PREVC, bug-fix

## Testes

Sempre adicionar testes para novos adapters/services/handlers.
