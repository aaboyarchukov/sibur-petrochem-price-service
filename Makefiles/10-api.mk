# Генерация кода из OpenAPI контракта.

OPENAPI_SPEC := api/openapi.yaml
OPENAPI_CONFIG := api/oapi-codegen.yaml
GENERATED_API_DIR := backend/internal/generated/api

.PHONY: api-generate
api-generate: ## Сгенерировать strict-server из OpenAPI спецификации
	@mkdir -p $(GENERATED_API_DIR)
	@$(OAPI_CODEGEN) -config $(OPENAPI_CONFIG) $(OPENAPI_SPEC)
	@echo "API code generated"

.PHONY: api-clean
api-clean: ## Удалить сгенерированный код API
	@rm -rf $(GENERATED_API_DIR)
	@echo "Generated API code removed"

.PHONY: api-regen
api-regen: ## Пересоздать код API (clean + generate)
	@$(MAKE) api-clean api-generate
