# Spec-Driven Development com Anyharness

Objetivo: conduzir o ciclo de implementacao seguindo a spec e usando o MCP local-harness com rastreabilidade completa (tools + sessions + contratos + evidencia).

## Quando usar

- Inicio de feature ou refactor com `spec_id` definido.
- Ajuste de bug quando houver risco de quebrar requisitos existentes.
- Fechamento de task com risco de scope change.

## Convenções obrigatorias

- Sempre iniciar com `session.start`.
- Sempre carregar a spec via `harness://contracts/specs/<spec_id>` antes de codar.
- Validar com `contract.task.next` para definir a proxima unidade de trabalho.
- Sempre registrar evidencias com `session.append`.
- Encerrar task somente via `contract.task.complete`.

## Fluxo recomendado (SDD local-harness)

### 1) Descoberta e baseline

- `judge.list()` para confirmar rubricas disponiveis.
- Ler guides relevantes:
  - `harness://guides/rules/<id>`
  - `harness://guides/skills/<id>`
  - `harness://guides/subagents/<id>`
- `sensor.list()` para mapear sensores por `regulation`:
  - `sensor.list({ regulation: "maintainability" })`
  - `sensor.list({ regulation: "fitness" })`
  - `sensor.list({ regulation: "behaviour" })`
- `session.start({ workflow: "PREVC", contract_id: "<spec_id>" })` ou workflow equivalente.
- `session.append({ session_id, event: { kind: "tool_call", tool: "judge.list", args: {}, timestamp } })`.
- `session.append({ session_id, event: { kind: "tool_call", tool: "sensor.list", args: { regulation: "maintainability" }, timestamp } })`.

### 2) Selecionar task da spec

- `contract.task.next({ spec_id })`.
- Se retornar `task: null`, encerre o fluxo de desenvolvimento (todas as tasks completas).
- Guarde `task_id`, `description` e `status`.
- Registre inicio da task:
  - `session.append({ session_id, event: { kind: "task_started", task_id: "<task_id>", timestamp } })`.

### 3) Validacao inicial por contrato

- Rode `contract.spec.validate({ id: "<spec_id>", artifact: "<alvo>" })`.
- Interprete a resposta:
  - `passed: true` e sem `pending[]`: segue para implementacao.
  - `pending` nao vazio: revisar cada entrada com `rubrica`.
  - sensoriamento parcial falhou: tratar como bug de produto, registrar e corrigir antes de codar.

### 4) Execucao de sensores (feedback computacional)

- Para cada sensor relevante da spec, rode:
  - `sensor.run({ id: "<sensor_id>", target: "<alvo>" })`.
- Use targets pequenos para ciclos rapidos (arquivo/folder especifico), depois amplie.
- Ao receber cada resultado, chame:
  - `session.append({ session_id, event: { kind: "sensor_run", sensor: "<sensor_id>", passed: <boolean>, violationCount: <qtd>, regulation: "<maintainability|fitness|behaviour>", timestamp } })`.

### 5) Execucao de judges (feedback inferencial)

- Para cada check de judge da spec **ou** pendencia de `contract.spec.validate`:
  - `judge.review({ rubric_id: "<rubric>", target: "<alvo>", spec_id: "<spec_id>" })`.
  - executar inferencia no cliente com `instructions/schema/context` retornados.
  - enviar resultado com:
    - `judge.record({ rubric_id: "<rubric>", target: "<alvo>", spec_id: "<spec_id>", result: <json_conforme_schema> })`.
- Registre em sessão:
  - `session.append({ session_id, event: { kind: "judge_review", rubric: "<rubric>", passed: <boolean>, violationCount: <qtd>, timestamp } })`.

### 6) Corrigir implementacao com escopo controlado

- Corrigir somente o necessario para satisfazer criterios da task.
- Nunca introduzir comportamento fora da spec sem atualizacao explicita da spec.
- Se houver ambiguidade critica, registre:
  - `session.append({ session_id, event: { kind: "decision", what: "espera alinhamento", rationale: "motivo", timestamp } })`.
- Opcionalmente use `session.append` com `human_intervention` se houver bloqueio por parte do humano.

### 7) Validacao final da task

- Repetir validacao de sensores afetados e rubricas pendentes.
- Rodar novamente:
  - `contract.spec.validate({ id: "<spec_id>", artifact: "<alvo>" })`.
- Se precisar retomar contexto de uma sessao existente: `session.get({ session_id })`.
- Considerar task pronta apenas com:
  - criterios da spec cobridos,
  - nenhum erro novo de escopo,
  - `contract.spec.validate` sem `pending` e com `passed: true`.

### 8) Fechar ciclo e evidencias

- Prepare evidence minimo (array com pelo menos um item):
  - `{ kind: "sensor_run", sensor: "tsc", passed: true, timestamp: "..." }`
  - `{ kind: "judge_review", rubric: "spec-adherence", passed: false, timestamp: "..." }`
  - `{ kind: "pr_link", url: "https://...", timestamp: "..." }`
  - `{ kind: "note", text: "decisao de escopo e razao", timestamp: "..." }`
- Chamar `contract.task.complete({ task_id, evidence: [ ... ] })`.
- Acrescentar log final:
  - `session.append({ session_id, event: { kind: "task_completed", task_id: "<task_id>", timestamp } })`.

## Uso de ferramentas nao comuns

- `sensor.register`: quando o projeto precisa de novo controle computacional repetivel.
  - Chamada obrigatoria com:
    - `id`, `kind`, `regulation`, `command`, `adapter`, opcionalmente `description` e `defaults`.
  - Exemplo minimo:
    - `sensor.register({"id":"infra-limits","kind":"custom","regulation":"fitness","command":"node scripts/check-infra-limits.js {target}","adapter":"passthrough","defaults":{"target":"src"}})`.
- `harness.steer.suggest`: usar por milestone ou fim de sprint para evoluir o harness.
  - `harness.steer.suggest({ windowDays: 30 })`.
  - Se sugestao vier com valor, proponha aprovacao humana e registre como tarefa futura.
- `session.get`: para recuperar contexto de continuidade quando retoma trabalho:
  - `session.get({ session_id })`.

## Checklist SDD (pre-close)

- [ ] `session.start` executado e session_id registrado.
- [ ] spec carregada corretamente via resource URI.
- [ ] `contract.task.next` retornou task concreta.
- [ ] `sensor.list` e `judge.list` revisados.
- [ ] sensores executados com `sensor.run` e falhas tratadas.
- [ ] judges executados via `judge.review` + `judge.record`.
- [ ] `contract.spec.validate` sem `pending[]` e `passed: true`.
- [ ] task encerrada por `contract.task.complete` com evidencias validas.
- [ ] opcional: `harness.steer.suggest` revisado para melhoria do harness.
