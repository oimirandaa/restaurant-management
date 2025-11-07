package controllers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/helpers"
	"restaurant-management-system/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
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
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		//Convert the JSON data coming from postman to something Golang can understand
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		//Validate the data based on user struct
		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		//You'll check if email has already been taken
		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking if email is already taken"})
			log.Panic(err)
			return
		}
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email is already taken"})
			return
		}

		//Hash the password
		password := HashPassword(*user.Password)
		user.Password = &password

		//You'll also check if the phone has already been used by another user
		count, err = userCollection.CountDocuments(ctx, bson.M{"phone": user.Phone})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking if phone is already taken"})
			log.Panic(err)
			return
		}
		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "phone number is already taken"})
			return
		}

		//Create some extra details for the user object - created_at, updated_at, id
		user.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.UserId = user.ID.Hex()

		//Generate Token and Refresh Token
		token, refreshToken, tokenErr := helpers.GenerateAllTokens(*user.Email, *user.FirstName, *user.LastName, user.UserId)
		if tokenErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while generating the token"})
			return
		}

		user.Token = &token
		user.RefreshToken = &refreshToken

		//Inserts the data into the database
		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			msg := fmt.Sprintf("error ocurred while inserting the user %s", insertErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		//Return the data
		c.JSON(http.StatusCreated, resultInsertionNumber)
	}
}

func LogIn() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User
		var foundUser models.User

		//Convert the login data from postman which is in JSON to Golang Readable format
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		//Find a user with the email, and if the user even exists
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while fetching the user"})
			return
		}

		//Then you will verify the password
		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		//If the password is correct, then you will generate a token and refresh token
		token, refreshToken, err := helpers.GenerateAllTokens(*foundUser.Email, *foundUser.FirstName, *foundUser.LastName, foundUser.UserId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while generating the token"})
			return
		}
		//Return the token and refresh token
		helpers.UpdateAllTokens(token, refreshToken, foundUser.UserId)

		//If the password is incorrect, then return an error
		c.JSON(http.StatusOK, foundUser)
	}
}

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Fatal(err)
	}

	return string(bytes)
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	bcryptErr := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	msg := ""
	check := true
	if bcryptErr != nil {
		msg = fmt.Sprintf("Password is incorrect")
		check = false
	}

	return check, msg
}
