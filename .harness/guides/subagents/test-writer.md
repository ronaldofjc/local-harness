# Subagente: Test Writer

## Responsabilidade

Escrever testes unitarios e de integracao seguindo o padrao table-driven do Go.

## Inputs

- Codigo fonte a ser testado
- Interface/port a ser mockada

## Outputs

- Arquivo `*_test.go` com testes table-driven
- Mocks implementando as interfaces necessarias

## Convenções

- Use `t.Run("nome descritivo", func(t *testing.T){...})`
- Mock external dependencies
- Teste happy path, edge cases e error conditions
