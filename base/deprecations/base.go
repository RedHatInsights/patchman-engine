package deprecations

import (
	"app/base/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

type Deprecation interface {
	Deprecate(*gin.Context)
}

//lint:ignore U1000 ignore unused, may be used in the future
type apiDeprecation struct {
	shouldDeprecate func(c *gin.Context) bool
	// datetime when API will be deprecated
	deprecationTimestamp time.Time
	// datetime when the deprecated api will be redirected
	redirectTimestamp *time.Time

	currentLocation  string
	locationReplacer *strings.Replacer
	redirectLocation *string

	message string
}

type limitDeprecation struct {
	shouldDeprecate      func(c *gin.Context) bool
	deprecationTimestamp time.Time
	message              string
}

func (d limitDeprecation) Deprecate(c *gin.Context) {
	if !utils.CoreCfg.LimitPageSize || !d.shouldDeprecate(c) {
		return
	}

	now := time.Now()
	httpDate := d.deprecationTimestamp.Format(time.RFC1123)
	setDeprecationHeader(c, httpDate, d.message)

	if now.Before(d.deprecationTimestamp) {
		// allow unlimited `limit`
		utils.CoreCfg.LimitPageSize = false
		c.Next()
		// reset LimitPageSize after all midllewares
		utils.CoreCfg.LimitPageSize = true
	}
}

func (d apiDeprecation) Deprecate(c *gin.Context) {
	if !d.shouldDeprecate(c) {
		return
	}

	now := time.Now()
	d.setDeprecationHeader(c)

	switch {
	case now.After(d.deprecationTimestamp):
		d.gone(c)
	case d.redirectTimestamp != nil && now.After(*d.redirectTimestamp):
		d.currentLocation = c.Request.URL.String()
		d.redirect(c)
	}
}

func (d *apiDeprecation) setDeprecationHeader(c *gin.Context) {
	// RFC1123 is HTTP-date format
	httpDate := d.deprecationTimestamp.Format(time.RFC1123)
	setDeprecationHeader(c, httpDate, d.message)
}

func (d *apiDeprecation) redirect(c *gin.Context) {
	if d.redirectLocation == nil {
		if d.locationReplacer == nil {
			utils.LogWarn("Ignoring Deprecation Redirect - one of `locationReplacer`, `redirectLocation` must be set")
			return
		}
		newLocation := d.locationReplacer.Replace(d.currentLocation)
		d.redirectLocation = &newLocation
	}
	// create redirect through c.Render because Timeout middleware can't handle code=-1 set by c.Redirect
	c.Render(http.StatusMovedPermanently, render.Redirect{
		Code:     http.StatusMovedPermanently,
		Location: *d.redirectLocation,
		Request:  c.Request,
	})
	c.Abort()
}

func (d *apiDeprecation) gone(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusGone, utils.ErrorResponse{Error: d.message})
}

func setDeprecationHeader(c *gin.Context, httpDate string, message string) {
	c.Header("Warning", message)

	// set Deprecation header
	// https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-deprecation-header
	c.Header("Deprecation", httpDate)

	// set Sunset header
	// https://datatracker.ietf.org/doc/html/rfc8594
	c.Header("Sunset", httpDate)
}
