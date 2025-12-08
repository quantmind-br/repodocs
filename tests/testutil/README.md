# Test Utilities Package

Este pacote contém utilitários e helpers para testes na aplicação repodocs-go.

## Componentes

### cache.go
Utilitários para testes de cache BadgerDB.

```go
// Criar cache em memória para teste
cache := testutil.NewBadgerCache(t)

// Verificar entrada de cache
testutil.VerifyCacheEntry(t, cache, "key", "value")
```

### temp.go
Gerenciamento de diretórios temporários para testes.

```go
// Criar diretório temporário
tmpDir := testutil.TempDir(t)

// Criar estrutura de saída temporária
baseDir, docsDir := testutil.TempOutputDir(t)
```

### http.go
Factory de servidor httptest com fixtures.

```go
// Criar servidor de teste
server := testutil.NewTestServer(t)

// Registrar handler
server.HandleHTML(t, "/test", "<html><body>Test</body></html>")
```

### logger.go
Loggers para testes.

```go
// Criar logger de teste
logger := testutil.NewTestLogger(t)

// Criar logger no-op
logger := testutil.NewNoOpLogger()
```

### documents.go
Factory de documentos para testes.

```go
// Criar documento de teste
doc := testutil.NewDocument(t)

// Criar documento HTML
doc := testutil.NewHTMLDocument(t, "https://example.com", "Title", "<html>...</html>")
```

### assertions.go
Asserções customizadas para documentos e arquivos.

```go
// Verificar conteúdo do documento
testutil.AssertDocumentContent(t, doc, "https://example.com", "Title", "Content")

// Verificar arquivo
testutil.AssertFileExists(t, "path/to/file")
testutil.AssertFileContains(t, "path/to/file", "expected content")
```

## Uso em Testes

```go
package unit

import (
    "testing"
    "github.com/quantmind-br/repodocs-go/tests/testutil"
)

func TestMyFunction(t *testing.T) {
    // Usar utilitários de teste
    cache := testutil.NewBadgerCache(t)
    server := testutil.NewTestServer(t)
    doc := testutil.NewDocument(t)

    // Executar testes
    // ...

    // Verificar resultados
    testutil.AssertDocumentContent(t, doc, expectedURL, expectedTitle, expectedContent)
}
```

## Mocks

Mocks são gerados automaticamente em `tests/mocks/domain.go` usando `mockgen`.

Para regenerar mocks:
```bash
mockgen -source=internal/domain/interfaces.go -destination=tests/mocks/domain.go -package mocks
```

## Fixtures

Fixtures de teste estão em `tests/testdata/fixtures/`:
- `strategies/` - HTML e XML para testes de estratégias
- `orchestrator/` - Configs e respostas para testes do orchestrator
- `renderer/` - HTML para testes de renderização
- `output/` - Arquivos de saída esperados (golden files)

## Configuração

Configs de teste estão em `tests/testdata/config/`:
- `test-config.yaml` - Config padrão
- `test-config-cache.yaml` - Config específico para cache
- `test-config-renderer.yaml` - Config específico para renderer
