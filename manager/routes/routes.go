package routes

import (
	"app/base/deprecations"
	"app/docs"
	"app/manager/controllers"
	"app/manager/middlewares"
	admin "app/turnpike/controllers"

	"github.com/gin-gonic/gin"
)

func InitAPI(api *gin.RouterGroup, config docs.EndpointsConfig) { // nolint: funlen
	api.Use(middlewares.CheckReferer())
	api.Use(middlewares.SetAPIVersion(api.BasePath()))
	api.Use(middlewares.Deprecate(deprecations.DeprecateLimit()))
	api.Use(middlewares.DatabaseWithContext())

	userAuth := api.Group("/")
	userAuth.Use(middlewares.RBAC())
	userAuth.Use(middlewares.PublicAuthenticator())

	systemAuth := api.Group("/")
	systemAuth.Use(middlewares.SystemCertAuthenticator())

	advisories := userAuth.Group("/advisories")
	advisories.GET("", controllers.AdvisoriesListHandler)
	advisories.GET("/:advisory_id", controllers.AdvisoryDetailHandler)
	advisories.GET("/:advisory_id/systems", controllers.AdvisorySystemsListHandler)

	if config.EnableBaselines {
		baselines := userAuth.Group("/baselines")
		baselines.GET("", controllers.BaselinesListHandler)
		baselines.GET("/:baseline_id", controllers.BaselineDetailHandler)
		baselines.GET("/:baseline_id/systems", controllers.BaselineSystemsListHandler)
		baselines.PUT("/", controllers.CreateBaselineHandler)
		baselines.PUT("/:baseline_id", controllers.BaselineUpdateHandler)
		baselines.DELETE("/:baseline_id", controllers.BaselineDeleteHandler)
		baselines.POST("/systems/remove", controllers.BaselineSystemsRemoveHandler)
	}

	systems := userAuth.Group("/systems")
	systems.GET("", controllers.SystemsListHandler)
	systems.GET("/:inventory_id", controllers.SystemDetailHandler)
	systems.GET("/:inventory_id/advisories", controllers.SystemAdvisoriesHandler)
	systems.GET("/:inventory_id/packages", controllers.SystemPackagesHandler)
	systems.GET("/:inventory_id/vmaas_json", controllers.SystemVmaasJSONHandler)
	systems.GET("/:inventory_id/yum_updates", controllers.SystemYumUpdatesHandler)
	systems.DELETE("/:inventory_id", controllers.SystemDeleteHandler)

	userAuth.GET("/tags", controllers.SystemTagListHandler)

	packages := userAuth.Group("/packages")
	packages.GET("", controllers.PackagesListHandler)
	packages.GET("/:package_name/systems", controllers.PackageSystemsListHandler)
	packages.GET("/:package_name/versions", controllers.PackageVersionsListHandler)
	packages.GET("/:package_name", controllers.PackageDetailHandler)

	export := userAuth.Group("export")
	export.GET("/advisories", controllers.AdvisoriesExportHandler)
	export.GET("/advisories/:advisory_id/systems", controllers.AdvisorySystemsExportHandler)

	export.GET("/systems", controllers.SystemsExportHandler)
	export.GET("/systems/:inventory_id/advisories", controllers.SystemAdvisoriesExportHandler)
	export.GET("/systems/:inventory_id/packages", controllers.SystemPackagesExportHandler)

	export.GET("/packages", controllers.PackagesExportHandler)
	export.GET("/packages/:package_name/systems", controllers.PackageSystemsExportHandler)
	if config.EnableBaselines {
		export.GET("/baselines/:baseline_id/systems", controllers.BaselineSystemsExportHandler)
	}
	if config.EnableTemplates {
		export.GET("/templates/:template_id/systems", controllers.TemplateSystemsExportHandler)
	}

	if config.EnableTemplates {
		templates := userAuth.Group("/templates")
		templates.GET("", controllers.TemplatesListHandler)
		templates.GET("/:template_id/systems", controllers.TemplateSystemsListHandler)
		templates.PATCH("/:template_id/systems", controllers.TemplateSystemsUpdateHandler)
		// update should be PATCH but keep PUT for backard compatibility
		templates.PUT("/:template_id/systems", controllers.TemplateSystemsUpdateHandler)
		templates.DELETE("/systems", controllers.TemplateSystemsDeleteHandler)

		systemTemplates := systemAuth.Group("/templates")
		systemTemplates.PATCH("/:template_id/subscribed-systems", controllers.TemplateSubscribedSystemsUpdateHandler)
	}

	views := userAuth.Group("/views")
	views.POST("/systems/advisories", controllers.PostSystemsAdvisories)
	views.POST("/advisories/systems", controllers.PostAdvisoriesSystems)

	ids := userAuth.Group("/ids")
	ids.GET("/advisories", controllers.AdvisoriesListIDsHandler)
	ids.GET("/advisories/:advisory_id/systems", controllers.AdvisorySystemsListIDsHandler)
	ids.GET("/packages/:package_name/systems", controllers.PackageSystemsListIDsHandler)
	ids.GET("/systems", controllers.SystemsListIDsHandler)
	ids.GET("/systems/:inventory_id/advisories", controllers.SystemAdvisoriesIDsHandler)
	if config.EnableBaselines {
		ids.GET("/baselines/:baseline_id/systems", controllers.BaselineSystemsListIDsHandler)
	}
	if config.EnableTemplates {
		ids.GET("/templates/:template_id/systems", controllers.TemplateSystemsListIDsHandler)
	}

	userAuth.GET("/status", controllers.Status)
}

func InitAdmin(app *gin.Engine, enableTurnpikeAuth bool) {
	api := app.Group("/api/patch/admin")
	if enableTurnpikeAuth {
		api.Use(middlewares.TurnpikeAuthenticator())
	}

	api.GET("/sync", admin.Syncapi)
	api.GET("/re-calc", admin.Recalc)
	api.GET("/check-caches", admin.CheckCaches)
	api.PUT("/refresh-packages", admin.RefreshPackagesHandler)
	api.PUT("/refresh-packages/:account", admin.RefreshPackagesAccountHandler)
	api.GET("/repack/:table_name", admin.RepackHandler)

	pprof := api.Group("/pprof")
	pprof.GET("/evaluator_upload/:param", admin.GetEvaluatorUploadPprof)
	pprof.GET("/evaluator_recalc/:param", admin.GetEvaluatorRecalcPprof)
	pprof.GET("/listener/:param", admin.GetListenerPprof)
	pprof.GET("/manager/:param", admin.GetManagerPprof)

	dbgroup := api.Group("/database")
	dbgroup.PUT("/pg_repack/recreate", admin.RepackRecreateHandler)
	dbgroup.GET("/sessions", admin.GetActiveSessionsHandler)
	dbgroup.GET("/sessions/:search", admin.GetActiveSessionsHandler)
	dbgroup.DELETE("/sessions/:pid", admin.TerminateSessionHandler)
}
