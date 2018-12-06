package vault

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/vault/helper/identity"
)

var templateStaticFuncs = map[string]interface{}{
	"replace":    strings.Replace,
	"has_prefix": strings.HasPrefix,
	"has_suffix": strings.HasSuffix,
	"contains":   strings.Contains,
}

var templateFuncFactories = map[string]func(ctx context.Context) interface{}{
	"identity": identityFuncFactory,
}

type policyTemplateContextKeys int

const (
	entityContextKey policyTemplateContextKeys = iota
	groupsContextKey
)

func templateFuncs(ctx context.Context) template.FuncMap {
	m := make(map[string]interface{}, len(templateFuncFactories)+len(templateStaticFuncs))
	for key, f := range templateStaticFuncs {
		m[key] = f
	}
	for key, f := range templateFuncFactories {
		m[key] = f(ctx)
	}
	return m
}

func identityFuncFactory(ctx context.Context) interface{} {
	entity, ok := ctx.Value(entityContextKey).(*identity.Entity)
	if !ok {
		entity = nil
	}
	groups, ok := ctx.Value(groupsContextKey).([]*identity.Group)
	if !ok {
		groups = nil
	}
	groupsMap := map[string]map[string]interface{}{}
	var entityMap map[string]interface{}
	var aliasMap map[string]map[string]interface{}
	if entity != nil {
		aliasMap = make(map[string]map[string]interface{}, len(entity.Aliases))
		for _, alias := range entity.Aliases {
			aliasMap[alias.MountAccessor] = map[string]interface{}{
				"id":       alias.ID,
				"name":     alias.Name,
				"metadata": alias.Metadata,
			}
		}
		entityMap = map[string]interface{}{
			"name":     entity.Name,
			"id":       entity.ID,
			"metadata": entity.Metadata,
			"aliases":  aliasMap,
		}
	}
	if groups != nil {
		groupsMap["ids"] = make(map[string]interface{}, len(groups))
		groupsMap["names"] = make(map[string]interface{}, len(groups))
		for _, group := range groups {
			groupMap := map[string]interface{}{
				"id":       group.ID,
				"name":     group.Name,
				"metadata": group.Metadata,
			}
			groupsMap["ids"][group.ID] = groupMap
			groupsMap["names"][group.Name] = groupMap

		}
	}
	identityMap := map[string]interface{}{
		"entity": entityMap,
		"groups": groupsMap,
	}
	return func() interface{} { return identityMap }
}

func renderPolicy(rules string, entity *identity.Entity, groups []*identity.Group) (string, bool, error) {
	var tpl *template.Template
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("use of invalid keyword leads to panic, check your policy")
		}
	}()

	ctx := context.Background()
	ctx = context.WithValue(ctx, entityContextKey, entity)
	ctx = context.WithValue(ctx, groupsContextKey, groups)
	tpl, err = template.New("").Funcs(templateFuncs(ctx)).Parse(rules)
	if err != nil {
		return "", false, errwrap.Wrapf("failed to template policy: {{err}}", err)
	}
	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, nil); err != nil {
		return "", false, errwrap.Wrapf("failed to execute the template: {{err}}", err)
	}

	retStr := buf.String()
	return retStr, len(retStr) != len(rules), err
}
