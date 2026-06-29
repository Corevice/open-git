package handler

import "github.com/labstack/echo/v4"

const isAdminContextKey = "is_admin"

func isSiteAdmin(c echo.Context) bool {
	v := c.Get(isAdminContextKey)
	if v == nil {
		return false
	}
	isAdmin, ok := v.(bool)
	return ok && isAdmin
}
