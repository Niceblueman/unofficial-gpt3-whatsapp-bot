package main

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/types"
	editType "github.com/GoAdminGroup/go-admin/template/types/table"
)

func GetKeytable(ctx *context.Context) (types.Panel, error) {
	api_keys := table.NewDefaultTable(table.DefaultConfigWithDriverAndConnection("sqlite", "apikeys"))
	info := api_keys.GetInfo()
	info.AddField("ID", "id", db.Int).FieldSortable()
	info.AddField("Key", "key", db.Varchar).FieldDisplay(func(value types.FieldModel) interface{} {
		return template.Default().
			Link().
			SetURL("/info/keys?__goadmin_detail_pk=" + value.Value).
			SetContent(template.HTML(value.Value)).
			OpenInNewTab().
			SetTabTitle(template.HTML("Key Detail(" + value.Value + ")")).
			GetContent()
	}).FieldSortable()
	info.AddField("Details", "details", db.Varchar).FieldEditAble(editType.Textarea).FieldSortable()
	info.AddField("Deadline", "deadline", db.Date).FieldSortable()
	info.SetTable("keys").SetTitle("Keys").SetDescription("Keys Management")
	return types.Panel{
		Content:     "<span class=\"logo-lg\"><b>Go</b>Admin</span>",
		Description: "Keys Management",
	}, nil
}
