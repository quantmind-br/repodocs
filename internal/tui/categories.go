package tui

type Category struct {
	ID            string
	Name          string
	Description   string
	Icon          string
	SubCategories []Category
}

var Categories = []Category{
	{ID: "output", Name: "Output", Description: "Output directory and format settings", Icon: ""},
	{ID: "exclude", Name: "Exclude Patterns", Description: "URL/path patterns to skip", Icon: ""},
	{ID: "concurrency", Name: "Concurrency", Description: "Workers, timeout, and depth limits", Icon: ""},
	{ID: "cache", Name: "Cache", Description: "Caching behavior and TTL", Icon: ""},
	{ID: "rendering", Name: "Rendering", Description: "JavaScript rendering options", Icon: ""},
	{ID: "stealth", Name: "Stealth", Description: "User-agent and delay settings", Icon: ""},
	{ID: "logging", Name: "Logging", Description: "Log level and format", Icon: ""},
	{ID: "llm", Name: "LLM", Description: "AI provider configuration", Icon: "", SubCategories: []Category{
		{ID: "llm_basic", Name: "Basic Settings", Description: "Provider, API key, and model", Icon: ""},
		{ID: "llm_rate_limit", Name: "Rate Limit", Description: "Request throttling settings", Icon: ""},
		{ID: "llm_circuit_breaker", Name: "Circuit Breaker", Description: "Failure protection settings", Icon: ""},
	}},
}

func (c *Category) HasSubCategories() bool {
	return len(c.SubCategories) > 0
}

func GetCategoryByID(id string) *Category {
	for i := range Categories {
		if Categories[i].ID == id {
			return &Categories[i]
		}
	}
	return nil
}

func GetCategoryNames() []string {
	names := make([]string, len(Categories))
	for i, c := range Categories {
		names[i] = c.Name
	}
	return names
}
