package controllers

import (
	"context"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "users")

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		recordsPerPage, err := strconv.Atoi(c.Query("recordsPerPage"))
		if err != nil || recordsPerPage < 1 {
			recordsPerPage = 10
		}

		page, err := strconv.Atoi(c.Query("page"))
		if err != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordsPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))
		if err != nil {
			startIndex = 0
		}

		matchStage := bson.D{
			{
				"$match", bson.D{},
			}}

		projectStage := bson.D{
			{
				"$project", bson.D{
					{"_id", 0},
					{"total_Count", 1},
					{"user_items", bson.D{
						{"$slice", []interface{}{"$data", startIndex, recordsPerPage}},
					}},
				},
			},
		}

		AggregationResult, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage,
			projectStage,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing the users"})
			return
		}

		var allUsers []models.User

		if err = AggregationResult.All(ctx, &allUsers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while listing the users"})
			return
		}

		c.JSON(http.StatusOK, allUsers[0])
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		userId := c.Param("user_id")
		var user models.User

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the user"})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		//Convert the JSON data coming from postman to something Golang can understand

		//Validate the data based on user struct

		//You'll check if email has already been taken

		//Hash the password

		//You'll also check if the phone has already been used by another user

		//Create some extra details for the user object - created_at, updated_at, id

		//Generate Token and Refresh Token

		//Inserts the data into the database

		//Return the data
	}
}

func LogIn() gin.HandlerFunc {
	return func(c *gin.Context) {
		//Convert the login data from postman which is in JSON to Golang Readable format

		//Find a user with the email, and if the user even exists

		//Then you will verify the password

		//If the password is correct, then you will generate a token and refresh token

		//Return the token and refresh token

		//If the password is incorrect, then return an error
	}
}

func HashPassword(password string) string {
	return ""
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	return false, ""
}
