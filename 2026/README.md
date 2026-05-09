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

### Entendendo os Resultados e a Avaliação da Rinha

A pontuação final da Rinha vai de **-6000 a +6000 pontos** e é a soma independente de dois fatores principais. O seu script de benchmark ajuda a medir a latência, que é metade do seu score:

1. **Score de Latência (`score_p99`):** 
   - A avaliação recompensa o seu **P99** logaritmicamente (cada 10x mais rápido = +1000 pontos). 
   - Se o seu P99 for **≤ 1ms**, você atinge o teto máximo de **+3000 pontos**.
   - Se o seu P99 passar de **2000ms**, você sofre o corte e recebe **-3000 pontos**.

2. **Score de Detecção (`score_det`):** 
   - Penaliza as previsões incorretas com diferentes pesos: Erros HTTP pesam muito (5), Falsos Negativos pesam médio (3) e Falsos Positivos pesam pouco (1).
   - Se a sua taxa bruta de falhas (FP + FN + HTTP Errors) passar de **15%**, seu score de detecção é cortado direto para **-3000 pontos**.

**Como melhorar sua pontuação:** O `SearchNeighbors` atual faz *brute-force* nos 3 milhões de vetores, resultando em alto consumo de CPU e P99 elevado. Para garantir os +3000 de latência sem perder a acurácia, você deverá substituir a força bruta por algoritmos de Busca Aproximada (ANN), como **HNSW**, **IVF**, ou buscas exatas como **VP Tree**. Além disso, prefira retornar uma resposta "chutada" rápida ao invés de devolver erro 500 caso ocorra falha interna!

## 4. Encerrando o ambiente

Para desligar o servidor e limpar a rede:

```bash
docker compose down
```
