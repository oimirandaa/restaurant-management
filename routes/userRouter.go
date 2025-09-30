package routes

import (
	"github.com/gin-gonic/gin"
	controller "restaurant-management-system/controllers"
)

func UserRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/users", controller.GetUsers())
	incomingRoutes.GET("users/:user_id", controller.GetUser())
	incomingRoutes.POST("users/signup", controller.SignUp())
	incomingRoutes.POST("users/signin", controller.SignIn())
}
