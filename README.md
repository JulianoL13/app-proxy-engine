# Proxy Engine

> [Read in English](README_EN.md)


Esse projeto surgiu de uma necessidade pessoal: tenho vários projetos de scraping em mente e já tive problemas com bloqueios de IP. Daí tive a ideia de criar uma ferramenta própria que agrega proxies gratuitos da internet.

Aproveitei para implementar em **Go**, linguagem que estou estudando. Também tentei aplicar conceitos de **Clean Architecture** (mesmo sendo overkill para o escopo, eu sei disso), além de outros aprendizados.

> **Aviso:** A API em si será privada para uso pessoal, mas o código é livre. **Use por sua conta e risco.**
> Planejo fazer uma verificação de segurança (conteúdo nocivo na resposta dos proxies), mas até lá, o risco existe.

Sinta-se livre para explorar o código.

---

## 🚀 Como Rodar

O jeito mais fácil é via Docker Compose, que já sobe o Redis e a API configurados.

### Pré-requisitos
- Docker & Docker Compose
- Go 1.22+ (para rodar localmente sem Docker)

### Rodando tudo (Infra + App)
```bash
docker-compose up --build
```

A API estará disponível em `http://localhost:8080`.

### Rodando localmente (Dev)
Se quiser rodar o Go na sua máquina e só o Redis no Docker:

1. Suba o Redis:
   ```bash
   make docker-redis
   ```
2. Rode a aplicação:
   ```bash
   make run
   # ou para desenvolvimento com hot-reload (se tiver air instalado)
   make dev
   ```

## 🛠 Comandos Úteis

O projeto possui um `Makefile` para facilitar a vida:

- `make test`: Roda os testes unitários
- `make lint`: Roda o linter (golangci-lint)
- `make check`: Limpa, gera mocks e roda os testes (combo completo)
