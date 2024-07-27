package controller

import (
	"context"
	"fmt"
	"golang-restaurant-management/database"
	helper "golang-restaurant-management/helpers"
	"golang-restaurant-management/models"
	"golang-restaurant-management/tasks"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user")
var Validate = validator.New()

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil || page < 1 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage
		startIndex, err = strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{{"$match", bson.D{{}}}}
		projectStage := bson.D{
			{"$project", bson.D{
				{"_id", 0},
				{"total_count", 1},
				{"user_items", bson.D{{"$slice", []interface{}{"$data", startIndex, recordPerPage}}}},
			}}}

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{
			matchStage, projectStage})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing user items"})
		}

		var allUsers []bson.M
		if err = result.All(ctx, &allUsers); err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, allUsers[0])

	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		userId := c.Param("user_id")

		var user models.User

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)

		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing user items"})
		}
		c.JSON(http.StatusOK, user)
	}
}

func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			log.Panic(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while checking for the email"})
			return
		}

		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "this email already exists"})
			return
		}

		// Hash password
		password := HashPassword(*user.Password)
		user.Password = &password

		// Generate OTP
		otp, err := tasks.GenerateOTP(*user.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
			return
		}

		// Store OTP
		err = tasks.StoreOTP(*user.Email, otp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
			return
		}

		// Queue verification email
		err = tasks.QueueVerificationEmail(*user.Email, otp)
		if err != nil {
			log.Printf("Error queueing verification email: %v", err)
		}

		user.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()

		resultInsertionNumber, insertErr := userCollection.InsertOne(ctx, user)
		if insertErr != nil {
			msg := fmt.Sprintf("User item was not created")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":    "User created successfully. Please check your email for verification.",
			"insertedID": resultInsertionNumber.InsertedID,
		})
	}
}

func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		var foundUser models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found, login seems to be incorrect"})
			return
		}

		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
		defer cancel()
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		if !foundUser.Is_Verified {
			c.JSON(http.StatusForbidden, gin.H{"error": "user not verified"})
			return
		}
		token, refreshToken, _ := helper.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, foundUser.User_id)

		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)

		// Create a new struct to hold the response data
		response := struct {
			ID            primitive.ObjectID `json:"id"`
			First_name    *string            `json:"first_name"`
			Last_name     *string            `json:"last_name"`
			Email         *string            `json:"email"`
			Phone         *string            `json:"phone"`
			Token         *string            `json:"token"`
			Refresh_Token *string            `json:"refresh_token"`
			User_id       string             `json:"user_id"`
			Is_Verified   bool               `json:"is_verified"`
		}{
			ID:            foundUser.ID,
			First_name:    foundUser.First_name,
			Last_name:     foundUser.Last_name,
			Email:         foundUser.Email,
			Phone:         foundUser.Phone,
			Token:         foundUser.Token,
			Refresh_Token: foundUser.Refresh_Token,
			User_id:       foundUser.User_id,
			Is_Verified:   foundUser.Is_Verified,
		}

		c.JSON(http.StatusOK, response)
	}
}

// func Login() gin.HandlerFunc {
// 	return func(c *gin.Context) {

// 		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
// 		var user models.User
// 		var foundUser models.User

// 		//convert the login data from postman which is in JSON to golang readable format

// 		if err := c.BindJSON(&user); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		//find a user with that email and see if that user even exists

// 		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
// 		defer cancel()
// 		if err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found, login seems to be incorrect"})
// 			return
// 		}

// 		//then you will verify the password

// 		passwordIsValid, msg := VerifyPassword(*user.Password, *foundUser.Password)
// 		defer cancel()
// 		if passwordIsValid != true {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
// 			return
// 		}

// 		//if all goes well, then you'll generate tokens

// 		token, refreshToken, _ := helper.GenerateAllTokens(*foundUser.Email, *foundUser.First_name, *foundUser.Last_name, foundUser.User_id)

// 		//update tokens - token and refersh token
// 		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)

// 		//return statusOK
// 		c.JSON(http.StatusOK, foundUser)
// 	}
// }

func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

func VerifyPassword(userPassword string, providedPassword string) (bool, string) {

	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = fmt.Sprintf("login or password is incorrect")
		check = false
	}
	return check, msg
}

func VerifyOTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var verificationData struct {
			Email string `json:"email" binding:"required,email"`
			OTP   string `json:"otp" binding:"required"`
		}

		if err := c.BindJSON(&verificationData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Retrieve stored OTP
		storedOTP, err := tasks.GetStoredOTP(verificationData.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve OTP"})
			return
		}

		// Compare OTPs
		if verificationData.OTP != storedOTP {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
			return
		}

		// Update user verification status
		update := bson.M{
			"$set": bson.M{
				"is_verified":  true,
				"is_otp_valid": false,
			},
		}

		_, err = userCollection.UpdateOne(
			ctx,
			bson.M{"email": verificationData.Email},
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user verification status"})
			return
		}

		// Clear stored OTP
		err = tasks.ClearStoredOTP(verificationData.Email)
		if err != nil {
			log.Printf("Error clearing stored OTP: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully"})
	}
}

func ForgotPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var user models.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if user exists
		var foundUser models.User
		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Generate OTP
		otp, err := tasks.GenerateOTP(*user.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
			return
		}

		// Store OTP
		err = tasks.StoreOTP(*user.Email, otp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store OTP"})
			return
		}

		// Queue reset password email
		err = tasks.QueueResetPasswordEmail(*user.Email, otp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to queue reset password email"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Reset password OTP sent to your email"})
	}
}

func ResetPassword() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		var resetData struct {
			Email       string `json:"email" binding:"required,email"`
			OTP         string `json:"otp" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.BindJSON(&resetData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify OTP
		storedOTP, err := tasks.GetStoredOTP(resetData.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
			return
		}

		if resetData.OTP != storedOTP {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid OTP"})
			return
		}

		// Hash new password
		hashedPassword := HashPassword(resetData.NewPassword)

		// Update password in database
		update := bson.M{
			"$set": bson.M{
				"password": hashedPassword,
			},
		}

		_, err = userCollection.UpdateOne(
			ctx,
			bson.M{"email": resetData.Email},
			update,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		// Clear stored OTP
		tasks.ClearStoredOTP(resetData.Email)

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
	}
}