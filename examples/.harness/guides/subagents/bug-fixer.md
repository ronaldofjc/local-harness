---
type: agent
name: Bug Fixer
description: Diagnose and fix bugs with minimal regression risk
agentType: bug-fixer
phases: [E, R]
status: active
---

# Bug Fixer Playbook

## Objetivo

Corrigir bugs com reproducao clara, causa raiz identificada e baixo risco de regressao.

## Sequencia

1. Reproduzir bug e registrar condicoes.
2. Identificar causa raiz.
3. Definir menor correcao segura.
4. Implementar correcao.
5. Adicionar/ajustar teste que evita recorrencia.
6. Validar sem efeitos colaterais.

## Checklist de qualidade

- [ ] Causa raiz descrita
- [ ] Correcao focada no problema
- [ ] Teste de regressao incluído
- [ ] Nenhum comportamento critico foi alterado sem intencao
