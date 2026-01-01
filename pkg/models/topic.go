package models

type Category struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	DefaultRSS   []string `json:"default_rss,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
}

var DefaultCategories = []Category{
	{
		Name:        "silicon-chips",
		DisplayName: "Silicon & Semiconductors",
		Description: "Chip fabrication, lithography, semiconductor industry news",
		Keywords:    []string{"semiconductor", "chip fabrication", "TSMC", "Intel", "Samsung foundry", "EUV lithography", "nm process node"},
		DefaultRSS:  []string{"https://semianalysis.com/feed/", "https://www.anandtech.com/rss/"},
	},
	{
		Name:        "rust-lang",
		DisplayName: "Rust Programming",
		Description: "Rust language updates, crates, ecosystem news",
		Keywords:    []string{"rust programming", "rust lang", "crates.io", "rust async", "rust embedded"},
	},
	{
		Name:        "llm-inference",
		DisplayName: "LLM & AI Inference",
		Description: "Large language models, inference optimization, AI deployment",
		Keywords:    []string{"LLM inference", "transformer optimization", "quantization", "GGUF", "vLLM", "TensorRT-LLM"},
	},
	{
		Name:        "webgpu",
		DisplayName: "WebGPU & Graphics",
		Description: "WebGPU, browser graphics, GPU compute on the web",
		Keywords:    []string{"WebGPU", "WGSL", "browser GPU", "web graphics"},
	},
	{
		Name:        "systems-programming",
		DisplayName: "Systems Programming",
		Description: "OS development, compilers, low-level programming",
		Keywords:    []string{"operating systems", "compiler design", "LLVM", "systems programming", "kernel"},
	},
	{
		Name:        "kubernetes",
		DisplayName: "Kubernetes & Cloud Native",
		Description: "K8s, containers, cloud-native infrastructure",
		Keywords:    []string{"kubernetes", "k8s", "containers", "cloud native", "CNCF"},
	},
}

func GetCategoryByName(name string) *Category {
	for _, cat := range DefaultCategories {
		if cat.Name == name {
			return &cat
		}
	}
	return nil
}
