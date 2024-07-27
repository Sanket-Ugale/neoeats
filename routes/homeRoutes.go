// File: routes/homeRoutes.go
package routes

import (
	controller "golang-restaurant-management/controllers"

	"github.com/gin-gonic/gin"
)

func HomeRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/", controller.Home())
	incomingRoutes.GET("/docs", controller.Documentation())

}
