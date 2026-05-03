---
type: agent
name: New Task
description: Start implementation tasks with structured context loading and clear done criteria
agentType: new-task
phases: [E]
status: active
---

# New Task Playbook

## Objetivo

Iniciar qualquer tarefa de execucao com contexto suficiente, plano curto e criterio de conclusao.

## Passos obrigatorios

1. Ler `AGENTS.md` e `.context/agents/README.md`.
2. Ler especificacao em `vault/20-projects/` (quando existir).
3. Identificar repo alvo em `projects/<nome>/`.
4. Carregar contexto local do repo (`AGENTS.md` e `.context/` internos, se existirem).
5. Definir escopo tecnico e passos de implementacao.

## Definition of done

- Requisito implementado conforme criterios de aceite.
- Testes/lint relevantes executados.
- Riscos e pendencias registrados.
- Aprendizado principal registrado no `vault/` quando aplicavel.
