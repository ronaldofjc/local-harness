# Skill: Clean Architecture em Go

## Praticas

1. **Handlers finos**: HTTP handlers apenas parseiam input e delegam para services.
2. **Services com regras**: Toda logica de negocio fica em services.
3. **Repositories como ports**: Interfaces em `port/`, implementacoes em `repository/`.
4. **DTOs com validacao**: Request/Response structs com tags de binding e metodo `Validate()`.

## Anti-patterns

- Nao coloque logica de negocio em handlers.
- Nao acesse banco diretamente de services (use ports).
- Nao ignore erros (use `fmt.Errorf("...: %w", err)`).
