package tui

type Category struct {
	ID          string
	Name        string
	Description string
	Icon        string
}

var Categories = []Category{
	{ID: "output", Name: "Output", Description: "Output directory and format settings", Icon: ""},
	{ID: "concurrency", Name: "Concurrency", Description: "Workers, timeout, and depth limits", Icon: ""},
	{ID: "cache", Name: "Cache", Description: "Caching behavior and TTL", Icon: ""},
	{ID: "rendering", Name: "Rendering", Description: "JavaScript rendering options", Icon: ""},
	{ID: "stealth", Name: "Stealth", Description: "User-agent and delay settings", Icon: ""},
	{ID: "logging", Name: "Logging", Description: "Log level and format", Icon: ""},
	{ID: "llm", Name: "LLM", Description: "AI provider configuration", Icon: ""},
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
