package strategies

// RustdocIndex represents the top-level rustdoc JSON structure
type RustdocIndex struct {
	Root            interface{}               `json:"root"` // Can be int or string depending on format version
	CrateVersion    string                    `json:"crate_version"`
	FormatVersion   int                       `json:"format_version"`
	IncludesPrivate bool                      `json:"includes_private"`
	Index           map[string]*RustdocItem   `json:"index"`
	Paths           map[string]*RustdocPath   `json:"paths"`
	ExternalCrates  map[string]*ExternalCrate `json:"external_crates"`
}

// RustdocItem represents a single item in the index
type RustdocItem struct {
	ID          interface{}            `json:"id"` // Can be int or string
	CrateID     int                    `json:"crate_id"`
	Name        *string                `json:"name"` // nullable
	Span        *RustdocSpan           `json:"span"`
	Visibility  interface{}            `json:"visibility"` // Can be string or object
	Docs        *string                `json:"docs"`       // nullable, MARKDOWN!
	Links       map[string]interface{} `json:"links"`      // name -> item ID (int or string)
	Attrs       []interface{}          `json:"attrs"`
	Deprecation *RustdocDeprecation    `json:"deprecation"`
	Inner       map[string]interface{} `json:"inner"` // Dynamic based on item type
}

// RustdocSpan represents source code location
type RustdocSpan struct {
	Filename string `json:"filename"`
	Begin    [2]int `json:"begin"` // [line, column]
	End      [2]int `json:"end"`
}

// RustdocPath represents a path reference
type RustdocPath struct {
	Path string      `json:"path"`
	ID   interface{} `json:"id"` // Can be int or string
	Args interface{} `json:"args"`
	Kind string      `json:"kind"` // In paths map: "struct", "enum", "trait", etc.
}

// RustdocDeprecation represents deprecation info
type RustdocDeprecation struct {
	Since string `json:"since"`
	Note  string `json:"note"`
}

// ExternalCrate represents an external crate reference
type ExternalCrate struct {
	Name        string `json:"name"`
	HTMLRootURL string `json:"html_root_url"`
}

// RustdocModule represents a module item (extracted from inner)
type RustdocModule struct {
	IsCrate    bool          `json:"is_crate"`
	Items      []interface{} `json:"items"` // Can be int or string IDs
	IsStripped bool          `json:"is_stripped"`
}

// RustdocFunction represents a function/method (extracted from inner)
type RustdocFunction struct {
	Sig      *RustdocFunctionSig `json:"sig"`
	Generics *RustdocGenerics    `json:"generics"`
	Header   *RustdocHeader      `json:"header"`
	HasBody  bool                `json:"has_body"`
}

// RustdocFunctionSig represents a function signature
type RustdocFunctionSig struct {
	Inputs     []interface{} `json:"inputs"` // [[name, type], ...]
	Output     interface{}   `json:"output"` // nullable (void), type object
	IsVariadic bool          `json:"is_c_variadic"`
}

// RustdocGenerics represents generic parameters
type RustdocGenerics struct {
	Params          []RustdocGenericParam `json:"params"`
	WherePredicates []interface{}         `json:"where_predicates"`
}

// RustdocGenericParam represents a generic parameter
type RustdocGenericParam struct {
	Name string      `json:"name"`
	Kind interface{} `json:"kind"`
}

// RustdocHeader represents function header attributes
type RustdocHeader struct {
	IsConst  bool   `json:"is_const"`
	IsUnsafe bool   `json:"is_unsafe"`
	IsAsync  bool   `json:"is_async"`
	ABI      string `json:"abi"`
}

// RustdocTrait represents a trait (extracted from inner)
type RustdocTrait struct {
	IsAuto          bool             `json:"is_auto"`
	IsUnsafe        bool             `json:"is_unsafe"`
	IsDynCompatible bool             `json:"is_dyn_compatible"`
	Items           []interface{}    `json:"items"` // Can be int or string IDs
	Generics        *RustdocGenerics `json:"generics"`
	Bounds          []interface{}    `json:"bounds"`
	Implementations []interface{}    `json:"implementations"` // Can be int or string IDs
}

// RustdocStruct represents a struct (extracted from inner)
type RustdocStruct struct {
	Kind     interface{}      `json:"kind"` // "unit", "tuple", or struct fields
	Generics *RustdocGenerics `json:"generics"`
	Impls    []interface{}    `json:"impls"` // Can be int or string IDs
}

// RustdocEnum represents an enum (extracted from inner)
type RustdocEnum struct {
	Variants         []interface{}    `json:"variants"` // Can be int or string IDs
	Generics         *RustdocGenerics `json:"generics"`
	Impls            []interface{}    `json:"impls"` // Can be int or string IDs
	VariantsStripped bool             `json:"variants_stripped"`
}

// RustdocImpl represents an impl block (extracted from inner)
type RustdocImpl struct {
	IsUnsafe        bool             `json:"is_unsafe"`
	Generics        *RustdocGenerics `json:"generics"`
	ProvidedMethods []string         `json:"provided_trait_methods"`
	Trait           interface{}      `json:"trait"` // nullable, RustdocPath-like
	For             interface{}      `json:"for"`   // Type
	Items           []interface{}    `json:"items"` // Can be int or string IDs
	IsNegative      bool             `json:"is_negative"`
	IsSynthetic     bool             `json:"is_synthetic"`
	BlanketImpl     interface{}      `json:"blanket_impl"` // nullable, Type
}

// RustdocUse represents a re-export (use statement)
type RustdocUse struct {
	Source string      `json:"source"`
	Name   string      `json:"name"`
	ID     interface{} `json:"id"` // nullable if external
	IsGlob bool        `json:"is_glob"`
}

// RustdocTypeAlias represents a type alias (extracted from inner)
type RustdocTypeAlias struct {
	Type     interface{}      `json:"type"`
	Generics *RustdocGenerics `json:"generics"`
}

// RustdocConstant represents a constant (extracted from inner)
type RustdocConstant struct {
	Type   interface{} `json:"type"`
	Const_ interface{} `json:"const"` // The constant expression/value
}

// RustdocStatic represents a static variable (extracted from inner)
type RustdocStatic struct {
	Type      interface{} `json:"type"`
	IsMutable bool        `json:"mutable"`
	Expr      string      `json:"expr"`
}

// RustdocMacro represents a macro (extracted from inner)
type RustdocMacro struct {
	Macro string `json:"macro"`
}

// RustdocAssocType represents an associated type in a trait
type RustdocAssocType struct {
	Generics *RustdocGenerics `json:"generics"`
	Bounds   []interface{}    `json:"bounds"`
	Type     interface{}      `json:"type"` // Default type if any
}

// RustdocAssocConst represents an associated constant in a trait
type RustdocAssocConst struct {
	Type  interface{} `json:"type"`
	Value *string     `json:"value"` // Default value if any
}

// RustdocVariant represents an enum variant (extracted from inner)
type RustdocVariant struct {
	Kind         interface{} `json:"kind"` // "plain", "tuple", "struct"
	Discriminant interface{} `json:"discriminant"`
}

// Helper functions to extract typed inner values

// GetModule extracts module data from an item's inner field
func (item *RustdocItem) GetModule() *RustdocModule {
	if item.Inner == nil {
		return nil
	}
	if moduleData, ok := item.Inner["module"]; ok {
		return parseModule(moduleData)
	}
	return nil
}

// GetFunction extracts function data from an item's inner field
func (item *RustdocItem) GetFunction() *RustdocFunction {
	if item.Inner == nil {
		return nil
	}
	if fnData, ok := item.Inner["function"]; ok {
		return parseFunction(fnData)
	}
	return nil
}

// GetTrait extracts trait data from an item's inner field
func (item *RustdocItem) GetTrait() *RustdocTrait {
	if item.Inner == nil {
		return nil
	}
	if traitData, ok := item.Inner["trait"]; ok {
		return parseTrait(traitData)
	}
	return nil
}

// GetStruct extracts struct data from an item's inner field
func (item *RustdocItem) GetStruct() *RustdocStruct {
	if item.Inner == nil {
		return nil
	}
	if structData, ok := item.Inner["struct"]; ok {
		return parseStruct(structData)
	}
	return nil
}

// GetEnum extracts enum data from an item's inner field
func (item *RustdocItem) GetEnum() *RustdocEnum {
	if item.Inner == nil {
		return nil
	}
	if enumData, ok := item.Inner["enum"]; ok {
		return parseEnum(enumData)
	}
	return nil
}

// GetImpl extracts impl data from an item's inner field
func (item *RustdocItem) GetImpl() *RustdocImpl {
	if item.Inner == nil {
		return nil
	}
	if implData, ok := item.Inner["impl"]; ok {
		return parseImpl(implData)
	}
	return nil
}

// GetUse extracts use/re-export data from an item's inner field
func (item *RustdocItem) GetUse() *RustdocUse {
	if item.Inner == nil {
		return nil
	}
	if useData, ok := item.Inner["use"]; ok {
		return parseUse(useData)
	}
	return nil
}

// GetTypeAlias extracts type alias data from an item's inner field
func (item *RustdocItem) GetTypeAlias() *RustdocTypeAlias {
	if item.Inner == nil {
		return nil
	}
	if taData, ok := item.Inner["type_alias"]; ok {
		return parseTypeAlias(taData)
	}
	return nil
}

// GetConstant extracts constant data from an item's inner field
func (item *RustdocItem) GetConstant() *RustdocConstant {
	if item.Inner == nil {
		return nil
	}
	if constData, ok := item.Inner["constant"]; ok {
		return parseConstant(constData)
	}
	return nil
}

// GetMacro extracts macro data from an item's inner field
func (item *RustdocItem) GetMacro() *RustdocMacro {
	if item.Inner == nil {
		return nil
	}
	if macroData, ok := item.Inner["macro"]; ok {
		return parseMacro(macroData)
	}
	return nil
}

// GetVariant extracts variant data from an item's inner field
func (item *RustdocItem) GetVariant() *RustdocVariant {
	if item.Inner == nil {
		return nil
	}
	if variantData, ok := item.Inner["variant"]; ok {
		return parseVariant(variantData)
	}
	return nil
}

// IsPublic returns true if the item has public visibility
func (item *RustdocItem) IsPublic() bool {
	if item.Visibility == nil {
		return false
	}
	switch v := item.Visibility.(type) {
	case string:
		return v == "public"
	case map[string]interface{}:
		// Restricted visibility like pub(crate)
		return false
	default:
		return false
	}
}

// GetItemType returns the type of item based on the inner field
func (item *RustdocItem) GetItemType() string {
	if item.Inner == nil {
		return "unknown"
	}
	for key := range item.Inner {
		return key
	}
	return "unknown"
}

// Parser helper functions

func parseModule(data interface{}) *RustdocModule {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	mod := &RustdocModule{}
	if v, ok := m["is_crate"].(bool); ok {
		mod.IsCrate = v
	}
	if v, ok := m["items"].([]interface{}); ok {
		mod.Items = v
	}
	if v, ok := m["is_stripped"].(bool); ok {
		mod.IsStripped = v
	}
	return mod
}

func parseFunction(data interface{}) *RustdocFunction {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	fn := &RustdocFunction{}
	if sigData, ok := m["sig"].(map[string]interface{}); ok {
		fn.Sig = parseFunctionSig(sigData)
	}
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		fn.Generics = parseGenerics(genData)
	}
	if headerData, ok := m["header"].(map[string]interface{}); ok {
		fn.Header = parseHeader(headerData)
	}
	if v, ok := m["has_body"].(bool); ok {
		fn.HasBody = v
	}
	return fn
}

func parseFunctionSig(data map[string]interface{}) *RustdocFunctionSig {
	sig := &RustdocFunctionSig{}
	if v, ok := data["inputs"].([]interface{}); ok {
		sig.Inputs = v
	}
	sig.Output = data["output"]
	if v, ok := data["is_c_variadic"].(bool); ok {
		sig.IsVariadic = v
	}
	return sig
}

func parseGenerics(data map[string]interface{}) *RustdocGenerics {
	gen := &RustdocGenerics{}
	if params, ok := data["params"].([]interface{}); ok {
		for _, p := range params {
			if pm, ok := p.(map[string]interface{}); ok {
				param := RustdocGenericParam{}
				if name, ok := pm["name"].(string); ok {
					param.Name = name
				}
				param.Kind = pm["kind"]
				gen.Params = append(gen.Params, param)
			}
		}
	}
	if wp, ok := data["where_predicates"].([]interface{}); ok {
		gen.WherePredicates = wp
	}
	return gen
}

func parseHeader(data map[string]interface{}) *RustdocHeader {
	header := &RustdocHeader{}
	if v, ok := data["is_const"].(bool); ok {
		header.IsConst = v
	}
	if v, ok := data["is_unsafe"].(bool); ok {
		header.IsUnsafe = v
	}
	if v, ok := data["is_async"].(bool); ok {
		header.IsAsync = v
	}
	if v, ok := data["abi"].(string); ok {
		header.ABI = v
	}
	return header
}

func parseTrait(data interface{}) *RustdocTrait {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	trait := &RustdocTrait{}
	if v, ok := m["is_auto"].(bool); ok {
		trait.IsAuto = v
	}
	if v, ok := m["is_unsafe"].(bool); ok {
		trait.IsUnsafe = v
	}
	if v, ok := m["is_dyn_compatible"].(bool); ok {
		trait.IsDynCompatible = v
	}
	if v, ok := m["items"].([]interface{}); ok {
		trait.Items = v
	}
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		trait.Generics = parseGenerics(genData)
	}
	if v, ok := m["bounds"].([]interface{}); ok {
		trait.Bounds = v
	}
	if v, ok := m["implementations"].([]interface{}); ok {
		trait.Implementations = v
	}
	return trait
}

func parseStruct(data interface{}) *RustdocStruct {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	s := &RustdocStruct{}
	s.Kind = m["kind"]
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		s.Generics = parseGenerics(genData)
	}
	if v, ok := m["impls"].([]interface{}); ok {
		s.Impls = v
	}
	return s
}

func parseEnum(data interface{}) *RustdocEnum {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	e := &RustdocEnum{}
	if v, ok := m["variants"].([]interface{}); ok {
		e.Variants = v
	}
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		e.Generics = parseGenerics(genData)
	}
	if v, ok := m["impls"].([]interface{}); ok {
		e.Impls = v
	}
	if v, ok := m["variants_stripped"].(bool); ok {
		e.VariantsStripped = v
	}
	return e
}

func parseImpl(data interface{}) *RustdocImpl {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	impl := &RustdocImpl{}
	if v, ok := m["is_unsafe"].(bool); ok {
		impl.IsUnsafe = v
	}
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		impl.Generics = parseGenerics(genData)
	}
	if v, ok := m["provided_trait_methods"].([]interface{}); ok {
		for _, s := range v {
			if str, ok := s.(string); ok {
				impl.ProvidedMethods = append(impl.ProvidedMethods, str)
			}
		}
	}
	impl.Trait = m["trait"]
	impl.For = m["for"]
	if v, ok := m["items"].([]interface{}); ok {
		impl.Items = v
	}
	if v, ok := m["is_negative"].(bool); ok {
		impl.IsNegative = v
	}
	if v, ok := m["is_synthetic"].(bool); ok {
		impl.IsSynthetic = v
	}
	impl.BlanketImpl = m["blanket_impl"]
	return impl
}

func parseUse(data interface{}) *RustdocUse {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	use := &RustdocUse{}
	if v, ok := m["source"].(string); ok {
		use.Source = v
	}
	if v, ok := m["name"].(string); ok {
		use.Name = v
	}
	use.ID = m["id"]
	if v, ok := m["is_glob"].(bool); ok {
		use.IsGlob = v
	}
	return use
}

func parseTypeAlias(data interface{}) *RustdocTypeAlias {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	ta := &RustdocTypeAlias{}
	ta.Type = m["type"]
	if genData, ok := m["generics"].(map[string]interface{}); ok {
		ta.Generics = parseGenerics(genData)
	}
	return ta
}

func parseConstant(data interface{}) *RustdocConstant {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	c := &RustdocConstant{}
	c.Type = m["type"]
	c.Const_ = m["const"]
	return c
}

func parseMacro(data interface{}) *RustdocMacro {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	macro := &RustdocMacro{}
	if v, ok := m["macro"].(string); ok {
		macro.Macro = v
	}
	return macro
}

func parseVariant(data interface{}) *RustdocVariant {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	v := &RustdocVariant{}
	v.Kind = m["kind"]
	v.Discriminant = m["discriminant"]
	return v
}
