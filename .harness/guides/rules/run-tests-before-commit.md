# Regra: Sempre rodar testes antes de commit

## Motivacao

Testes falhos mergeados na main quebram o CI e bloqueiam outros desenvolvedores.

## Regra

Antes de qualquer commit, execute:

```bash
go test ./...
```

## Consequencias

- Commits com testes falhos sao revertidos automaticamente.
- O harness `go-test` ja cobre este check.
