package main

import (
	"html/template"
	"time"

	_ "github.com/GoAdminGroup/go-admin/adapter/gin"               // Import the adapter, it must be imported. If it is not imported, you need to define it yourself.
	_ "github.com/GoAdminGroup/go-admin/modules/db/drivers/sqlite" // Import the sql driver
	_ "github.com/GoAdminGroup/themes/adminlte"                    // Import the theme

	"github.com/GoAdminGroup/go-admin/engine"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/language"
	"github.com/gin-gonic/gin"
)

func useAdmin(r *gin.Engine) {
	// Instantiate a GoAdmin engine object.
	eng := engine.Default()
	// GoAdmin global configuration, can also be imported as a json file.
	cfg := config.Config{
		Databases: config.DatabaseList{
			"default": {
				Name:            "WhatsappApiAdmin",
				MaxIdleConns:    50,
				MaxOpenConns:    150,
				ConnMaxLifetime: time.Hour,
				Driver:          config.DriverSqlite,
				File:            "./admin.db",
			},
			"apikeys": {
				Name:            "Apikeys",
				MaxIdleConns:    50,
				MaxOpenConns:    150,
				ConnMaxLifetime: time.Hour,
				Driver:          config.DriverSqlite,
				File:            "./api_keys.db",
			},
		},
		// Store must be set and guaranteed to have write access, otherwise new administrator users cannot be added.
		Store: config.Store{
			Path:   "./uploads",
			Prefix: "uploads",
		},
		UrlPrefix: "admin",
		Logo:      template.HTML("<a href=\"/\" class=\"logo\"><img src=\"https://res.cloudinary.com/duzlh0xen/image/upload/v1687399177/icon-512-2_dragged_cyh0zp.png\" width=\"40\" height=\"40\"><span><strong>Wa</strong>admin</span></a>"),
		MiniLogo:  template.HTML("<img src=\"https://res.cloudinary.com/duzlh0xen/image/upload/v1687399177/icon-512-2_dragged_cyh0zp.png\" width=\"30\" height=\"30\">"),
		Domain:    "localhost:8385",
		Language:  language.EN,
	}
	cfg.OpenAdminApi = true
	// Add configuration and plugins, use the Use method to mount to the web framework.
	_ = eng.AddConfig(&cfg).Use(r)
	eng.HTML("GET", "/info/keys", GetKeytable)
}
