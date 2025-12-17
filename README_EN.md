# Proxy Engine

> [Ler em Português](README.md)

This project was born out of a personal need: I have several scraping projects in mind and have faced IP blocking issues in the past. So, I decided to build my own tool to aggregate free proxies from the internet.

I took this opportunity to implement it in **Go**, a language I am currently studying. I also tried to apply **Clean Architecture** concepts (I know it might be overkill for this scope), among other things I'm learning.

> **Warning:** The API itself will be private for my personal use, but the code is open source. **Use it at your own risk.**
> I plan to implement security checks (detecting harmful content in proxy responses), but until then, the risk exists.

Feel free to explore the code.

---

## 🚀 How to Run

The easiest way is using Docker Compose, which brings up both Redis and the API already configured.

### Prerequisites
- Docker & Docker Compose
- Go 1.22+ (to run locally without Docker)

### Running everything (Infra + App)
```bash
docker-compose up --build
```

The API will be available at `http://localhost:8080`.

### Running locally (Dev)
If you prefer running Go on your host machine and only Redis in Docker:

1. Start Redis:
   ```bash
   make docker-redis
   ```
2. Run the application:
   ```bash
   make run
   # or for development with hot-reload (if you have air installed)
   make dev
   ```

## 🛠 Useful Commands

The project includes a `Makefile` to make things easier:

- `make test`: Runs unit tests
- `make lint`: Runs the linter (golangci-lint)
- `make check`: Cleans, generates mocks, and runs tests (full combo)
