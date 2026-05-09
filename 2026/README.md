# Rinha de Backend 2026 - Testes e Benchmark

Este guia explica como subir o ambiente localmente, verificar os endpoints da API e realizar os testes de stress (benchmark) para avaliar a performance da solução.

## Pré-requisitos

- **Docker** e **Docker Compose** instalados.
- **Go 1.22+** instalado (opcional, apenas para rodar o benchmark local).

## 1. Subindo a Aplicação

A arquitetura usa o Docker Compose para subir:
- 1x **HAProxy** (Load Balancer exposto na porta `9999`)
- 2x **APIs Go** (Limitadas a 0.45 CPU e 170MB de RAM cada)

Para compilar as imagens e iniciar os containers em background, rode na raiz da pasta `2026/`:

```bash
docker compose up --build -d
```

Verifique se todos os containers iniciaram e carregaram o dataset corretamente (os logs devem mostrar `Engine loaded with 3000000 records`):

```bash
docker compose logs -f
```

## 2. Testando a API manualmente

Com os containers rodando, o HAProxy estará escutando na porta `9999`.

### Verificando o Health Check

A rota de prontidão garante que a API carregou todos os dados na memória:

```bash
curl -i http://localhost:9999/ready
```
*Deve retornar um `HTTP 200 OK`.*

### Testando a Detecção de Fraude (Score)

Para testar o endpoint principal e o cálculo de busca vetorial (KNN), você pode disparar um dos payloads de exemplo que o próprio repositório disponibiliza:

```bash
curl -s -X POST http://localhost:9999/fraud-score \
  -H "Content-Type: application/json" \
  -d "$(jq '.[0]' resources/example-payloads.json)"
```

*Exemplo de retorno esperado:*
```json
{"approved":true,"fraud_score":0}
```

## 3. Teste de Stress (Benchmark)

A Rinha avalia fortemente a latência da sua API. Para rodar um benchmark rápido focando em testes de estresse para a rota `/fraud-score`, adicionamos um script em Go que lê múltiplos payloads e os dispara simultaneamente contra o load balancer.

Se você tiver o Go instalado na máquina host, você pode rodar:

```bash
# Rodar 100 requisições com 4 usuários (conexões) simultâneas:
go run benchmark.go -n 100 -c 4
```

```bash
# Teste mais agressivo (ex: 1000 requisições, 20 usuários simultâneos)
go run benchmark.go -n 1000 -c 20
```

### Entendendo os Resultados

O script retornará os seguintes dados vitais para a Rinha:
- **Throughput (req/s):** Capacidade de vazão. O `SearchNeighbors` atual faz *brute-force* em 3 milhões de registros, por isso é normal esse número girar abaixo de 10 req/s.
- **Distribuição de Latência (P50, P90, P99):** O score final de latência da Rinha usa fortemente o **P99**.

Se o percentil **P99** da latência estiver acima de `2000 ms`, sua pontuação de performance afundará para `-3000`. Otimizações na lógica de busca e no motor de KNN são essenciais para subir o ranking!

## 4. Encerrando o ambiente

Para desligar o servidor e limpar a rede:

```bash
docker compose down
```
