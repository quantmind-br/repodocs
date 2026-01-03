package strategies

import (
	"fmt"
	"strings"
)

type RustdocRenderer struct {
	index     *RustdocIndex
	crateName string
	version   string
}

func NewRustdocRenderer(index *RustdocIndex, crateName, version string) *RustdocRenderer {
	return &RustdocRenderer{
		index:     index,
		crateName: crateName,
		version:   version,
	}
}

func (r *RustdocRenderer) RenderItem(item *RustdocItem) string {
	var sb strings.Builder

	itemType := r.getItemType(item)
	name := ""
	if item.Name != nil {
		name = *item.Name
	}

	if name != "" {
		sb.WriteString(fmt.Sprintf("# %s `%s`\n\n", itemType, name))
	}

	if item.Deprecation != nil {
		sb.WriteString("> **Deprecated**")
		if item.Deprecation.Since != "" {
			sb.WriteString(fmt.Sprintf(" since %s", item.Deprecation.Since))
		}
		if item.Deprecation.Note != "" {
			sb.WriteString(fmt.Sprintf(": %s", item.Deprecation.Note))
		}
		sb.WriteString("\n\n")
	}

	sig := r.renderSignature(item)
	if sig != "" {
		sb.WriteString("```rust\n")
		sb.WriteString(sig)
		sb.WriteString("\n```\n\n")
	}

	if item.Docs != nil && *item.Docs != "" {
		docs := r.resolveCrossRefs(*item.Docs, item.Links)
		sb.WriteString(docs)
		sb.WriteString("\n\n")
	}

	if mod := item.GetModule(); mod != nil {
		sb.WriteString(r.renderModuleContents(item))
	}

	if trait := item.GetTrait(); trait != nil {
		sb.WriteString(r.renderTraitContents(item))
	}

	if item.GetStruct() != nil || item.GetEnum() != nil {
		sb.WriteString(r.renderImplContents(item))
	}

	return sb.String()
}

func (r *RustdocRenderer) getItemType(item *RustdocItem) string {
	if mod := item.GetModule(); mod != nil {
		if mod.IsCrate {
			return "Crate"
		}
		return "Module"
	}
	if item.GetStruct() != nil {
		return "Struct"
	}
	if item.GetEnum() != nil {
		return "Enum"
	}
	if item.GetTrait() != nil {
		return "Trait"
	}
	if item.GetFunction() != nil {
		return "Function"
	}
	if item.GetTypeAlias() != nil {
		return "Type Alias"
	}
	if item.GetConstant() != nil {
		return "Constant"
	}
	if item.GetMacro() != nil {
		return "Macro"
	}
	if item.GetUse() != nil {
		return "Re-export"
	}
	if item.GetVariant() != nil {
		return "Variant"
	}
	return "Item"
}

func (r *RustdocRenderer) renderSignature(item *RustdocItem) string {
	if item.GetFunction() != nil {
		return r.renderFunctionSignature(item)
	}
	if item.GetTrait() != nil {
		return r.renderTraitSignature(item)
	}
	if item.GetStruct() != nil {
		return r.renderStructSignature(item)
	}
	if item.GetEnum() != nil {
		return r.renderEnumSignature(item)
	}
	if item.GetTypeAlias() != nil {
		return r.renderTypeAliasSignature(item)
	}
	if item.GetConstant() != nil {
		return r.renderConstantSignature(item)
	}
	return ""
}

func (r *RustdocRenderer) renderFunctionSignature(item *RustdocItem) string {
	fn := item.GetFunction()
	if fn == nil {
		return ""
	}

	var sb strings.Builder

	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	if fn.Header != nil {
		if fn.Header.IsConst {
			sb.WriteString("const ")
		}
		if fn.Header.IsAsync {
			sb.WriteString("async ")
		}
		if fn.Header.IsUnsafe {
			sb.WriteString("unsafe ")
		}
	}

	sb.WriteString("fn ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}

	if fn.Generics != nil {
		sb.WriteString(r.renderGenerics(fn.Generics))
	}

	sb.WriteString("(")
	if fn.Sig != nil {
		for i, input := range fn.Sig.Inputs {
			if i > 0 {
				sb.WriteString(", ")
			}
			if arr, ok := input.([]interface{}); ok && len(arr) >= 2 {
				name := fmt.Sprintf("%v", arr[0])
				typeStr := r.RenderType(arr[1])
				if name == "self" {
					sb.WriteString(typeStr)
				} else {
					sb.WriteString(fmt.Sprintf("%s: %s", name, typeStr))
				}
			}
		}
	}
	sb.WriteString(")")

	if fn.Sig != nil && fn.Sig.Output != nil {
		outputStr := r.RenderType(fn.Sig.Output)
		if outputStr != "" && outputStr != "()" {
			sb.WriteString(" -> ")
			sb.WriteString(outputStr)
		}
	}

	if fn.Generics != nil {
		sb.WriteString(r.renderWhereClauses(fn.Generics))
	}

	return sb.String()
}

func (r *RustdocRenderer) RenderType(t interface{}) string {
	if t == nil {
		return "()"
	}

	switch v := t.(type) {
	case map[string]interface{}:
		return r.RenderTypeMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (r *RustdocRenderer) RenderTypeMap(t map[string]interface{}) string {
	if prim, ok := t["primitive"]; ok {
		return fmt.Sprintf("%v", prim)
	}

	if gen, ok := t["generic"]; ok {
		return fmt.Sprintf("%v", gen)
	}

	if resolved, ok := t["resolved_path"].(map[string]interface{}); ok {
		path := fmt.Sprintf("%v", resolved["path"])
		if args := resolved["args"]; args != nil {
			if argsMap, ok := args.(map[string]interface{}); ok {
				if angleArgs, ok := argsMap["angle_bracketed"].(map[string]interface{}); ok {
					if typeArgs, ok := angleArgs["args"].([]interface{}); ok && len(typeArgs) > 0 {
						var argStrs []string
						for _, arg := range typeArgs {
							if argMap, ok := arg.(map[string]interface{}); ok {
								if typeArg, ok := argMap["type"]; ok {
									argStrs = append(argStrs, r.RenderType(typeArg))
								}
							}
						}
						if len(argStrs) > 0 {
							path += "<" + strings.Join(argStrs, ", ") + ">"
						}
					}
				}
			}
		}
		return path
	}

	if borrowed, ok := t["borrowed_ref"].(map[string]interface{}); ok {
		mut := ""
		if borrowed["is_mutable"] == true {
			mut = "mut "
		}
		lifetime := ""
		if l, ok := borrowed["lifetime"].(string); ok && l != "" {
			lifetime = l + " "
		}
		inner := r.RenderType(borrowed["type"])
		return fmt.Sprintf("&%s%s%s", lifetime, mut, inner)
	}

	if slice, ok := t["slice"]; ok {
		return fmt.Sprintf("[%s]", r.RenderType(slice))
	}

	if arr, ok := t["array"].(map[string]interface{}); ok {
		innerType := r.RenderType(arr["type"])
		length := arr["len"]
		return fmt.Sprintf("[%s; %v]", innerType, length)
	}

	if tuple, ok := t["tuple"].([]interface{}); ok {
		if len(tuple) == 0 {
			return "()"
		}
		parts := make([]string, len(tuple))
		for i, elem := range tuple {
			parts[i] = r.RenderType(elem)
		}
		return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
	}

	if rawPtr, ok := t["raw_pointer"].(map[string]interface{}); ok {
		mut := "*const"
		if rawPtr["is_mutable"] == true {
			mut = "*mut"
		}
		inner := r.RenderType(rawPtr["type"])
		return fmt.Sprintf("%s %s", mut, inner)
	}

	if implTrait, ok := t["impl_trait"].([]interface{}); ok {
		var bounds []string
		for _, bound := range implTrait {
			if boundMap, ok := bound.(map[string]interface{}); ok {
				if traitBound, ok := boundMap["trait_bound"].(map[string]interface{}); ok {
					if trait, ok := traitBound["trait"].(map[string]interface{}); ok {
						if path, ok := trait["path"].(string); ok {
							bounds = append(bounds, path)
						}
					}
				}
			}
		}
		if len(bounds) > 0 {
			return "impl " + strings.Join(bounds, " + ")
		}
		return "impl ..."
	}

	if qualPath, ok := t["qualified_path"].(map[string]interface{}); ok {
		name := ""
		if n, ok := qualPath["name"].(string); ok {
			name = n
		}
		return name
	}

	return "..."
}

func (r *RustdocRenderer) renderGenerics(g *RustdocGenerics) string {
	if g == nil || len(g.Params) == 0 {
		return ""
	}

	var parts []string
	for _, p := range g.Params {
		if p.Name != "" && p.Name != "Self" {
			parts = append(parts, p.Name)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("<%s>", strings.Join(parts, ", "))
}

func (r *RustdocRenderer) renderWhereClauses(g *RustdocGenerics) string {
	if g == nil || len(g.WherePredicates) == 0 {
		return ""
	}
	return ""
}

func (r *RustdocRenderer) resolveCrossRefs(docs string, links map[string]interface{}) string {
	if links == nil || len(links) == 0 {
		return docs
	}

	result := docs
	for name, id := range links {
		targetItem := r.getItemByID(id)
		if targetItem == nil {
			continue
		}

		targetName := ""
		if targetItem.Name != nil {
			targetName = *targetItem.Name
		}

		targetURL := fmt.Sprintf("https://docs.rs/%s/%s/%s/%s",
			r.crateName, r.version, r.crateName, targetName)

		cleanName := strings.Trim(name, "`")
		result = strings.ReplaceAll(result,
			fmt.Sprintf("[%s]", name),
			fmt.Sprintf("[%s](%s)", cleanName, targetURL))
	}

	return result
}

func (r *RustdocRenderer) getItemByID(id interface{}) *RustdocItem {
	if r.index == nil {
		return nil
	}
	switch v := id.(type) {
	case string:
		return r.index.Index[v]
	case float64:
		return r.index.Index[fmt.Sprintf("%.0f", v)]
	case int:
		return r.index.Index[fmt.Sprintf("%d", v)]
	default:
		return nil
	}
}

func (r *RustdocRenderer) renderModuleContents(item *RustdocItem) string {
	mod := item.GetModule()
	if mod == nil || len(mod.Items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Contents\n\n")

	groups := make(map[string][]*RustdocItem)
	for _, childID := range mod.Items {
		child := r.getItemByID(childID)
		if child == nil || child.Name == nil {
			continue
		}
		itemType := r.getItemType(child)
		groups[itemType] = append(groups[itemType], child)
	}

	order := []string{"Module", "Struct", "Enum", "Trait", "Function", "Type Alias", "Constant", "Macro"}
	for _, itemType := range order {
		if items, ok := groups[itemType]; ok && len(items) > 0 {
			sb.WriteString(fmt.Sprintf("### %ss\n\n", itemType))
			for _, child := range items {
				if child.Name != nil {
					sb.WriteString(fmt.Sprintf("- `%s`\n", *child.Name))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (r *RustdocRenderer) renderTraitContents(item *RustdocItem) string {
	trait := item.GetTrait()
	if trait == nil || len(trait.Items) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Required Methods\n\n")

	for _, childID := range trait.Items {
		child := r.getItemByID(childID)
		if child == nil {
			continue
		}

		if fn := child.GetFunction(); fn != nil && child.Name != nil {
			sb.WriteString(fmt.Sprintf("### `%s`\n\n", *child.Name))
			sb.WriteString("```rust\n")
			sb.WriteString(r.renderFunctionSignature(child))
			sb.WriteString("\n```\n\n")
			if child.Docs != nil {
				sb.WriteString(*child.Docs)
				sb.WriteString("\n\n")
			}
		}
	}

	return sb.String()
}

func (r *RustdocRenderer) renderImplContents(item *RustdocItem) string {
	var impls []interface{}
	if st := item.GetStruct(); st != nil {
		impls = st.Impls
	} else if en := item.GetEnum(); en != nil {
		impls = en.Impls
	}

	if len(impls) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Implementations\n\n")

	for _, implID := range impls {
		implItem := r.getItemByID(implID)
		if implItem == nil {
			continue
		}

		impl := implItem.GetImpl()
		if impl == nil {
			continue
		}

		if impl.Trait != nil {
			if traitPath, ok := impl.Trait.(map[string]interface{}); ok {
				if path, ok := traitPath["path"].(string); ok {
					sb.WriteString(fmt.Sprintf("### impl %s\n\n", path))
				}
			}
		} else {
			sb.WriteString("### impl\n\n")
		}

		for _, methodID := range impl.Items {
			method := r.getItemByID(methodID)
			if method == nil || method.Name == nil {
				continue
			}

			sb.WriteString(fmt.Sprintf("#### `%s`\n\n", *method.Name))
			if fn := method.GetFunction(); fn != nil {
				sb.WriteString("```rust\n")
				sb.WriteString(r.renderFunctionSignature(method))
				sb.WriteString("\n```\n\n")
			}
			if method.Docs != nil {
				sb.WriteString(*method.Docs)
				sb.WriteString("\n\n")
			}
		}
	}

	return sb.String()
}

func (r *RustdocRenderer) renderTraitSignature(item *RustdocItem) string {
	trait := item.GetTrait()
	if trait == nil {
		return ""
	}

	var sb strings.Builder
	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	if trait.IsUnsafe {
		sb.WriteString("unsafe ")
	}
	if trait.IsAuto {
		sb.WriteString("auto ")
	}
	sb.WriteString("trait ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}
	if trait.Generics != nil {
		sb.WriteString(r.renderGenerics(trait.Generics))
	}

	return sb.String()
}

func (r *RustdocRenderer) renderStructSignature(item *RustdocItem) string {
	st := item.GetStruct()
	if st == nil {
		return ""
	}

	var sb strings.Builder
	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	sb.WriteString("struct ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}
	if st.Generics != nil {
		sb.WriteString(r.renderGenerics(st.Generics))
	}

	return sb.String()
}

func (r *RustdocRenderer) renderEnumSignature(item *RustdocItem) string {
	en := item.GetEnum()
	if en == nil {
		return ""
	}

	var sb strings.Builder
	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	sb.WriteString("enum ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}
	if en.Generics != nil {
		sb.WriteString(r.renderGenerics(en.Generics))
	}

	return sb.String()
}

func (r *RustdocRenderer) renderTypeAliasSignature(item *RustdocItem) string {
	ta := item.GetTypeAlias()
	if ta == nil {
		return ""
	}

	var sb strings.Builder
	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	sb.WriteString("type ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}
	if ta.Generics != nil {
		sb.WriteString(r.renderGenerics(ta.Generics))
	}
	if ta.Type != nil {
		sb.WriteString(" = ")
		sb.WriteString(r.RenderType(ta.Type))
	}

	return sb.String()
}

func (r *RustdocRenderer) renderConstantSignature(item *RustdocItem) string {
	c := item.GetConstant()
	if c == nil {
		return ""
	}

	var sb strings.Builder
	if item.IsPublic() {
		sb.WriteString("pub ")
	}
	sb.WriteString("const ")
	if item.Name != nil {
		sb.WriteString(*item.Name)
	}
	if c.Type != nil {
		sb.WriteString(": ")
		sb.WriteString(r.RenderType(c.Type))
	}

	return sb.String()
}
