package tasks

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client

func init() {
	redisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"), // Update this with your Redis server address
	})
}

func GenerateOTP(email string) (string, error) {
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	return otp, nil
}

func StoreOTP(email, otp string) error {
	ctx := context.Background()
	err := redisClient.Set(ctx, email, otp, 15*time.Minute).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetStoredOTP(email string) (string, error) {
	ctx := context.Background()
	otp, err := redisClient.Get(ctx, email).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("OTP not found for email: %s", email)
		}
		return "", err
	}
	return otp, nil
}

func ClearStoredOTP(email string) error {
	ctx := context.Background()
	err := redisClient.Del(ctx, email).Err()
	if err != nil {
		return err
	}
	return nil
}

// // tasks/otp.go
// package tasks

// import (
// 	"context"
// 	"fmt"
// 	"math/rand"
// 	"time"

// 	"go.mongodb.org/mongo-driver/bson"
// 	"go.mongodb.org/mongo-driver/mongo"
// )

// var userCollection *mongo.Collection

// func SetUserCollection(collection *mongo.Collection) {
// 	userCollection = collection
// }

// func GenerateOTP(email string) (string, error) {
// 	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
// 	return otp, nil
// }

// func StoreOTP(email, otp string) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	filter := bson.M{"email": email}
// 	update := bson.M{
// 		"$set": bson.M{
// 			"otp":          otp,
// 			"is_otp_valid": true,
// 			"updated_at":   time.Now(),
// 		},
// 	}

// 	_, err := userCollection.UpdateOne(ctx, filter, update)
// 	return err
// }

// func GetStoredOTP(email string) (string, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	var user struct {
// 		OTP        string `bson:"otp"`
// 		IsOTPValid bool   `bson:"is_otp_valid"`
// 	}

// 	filter := bson.M{"email": email}
// 	err := userCollection.FindOne(ctx, filter).Decode(&user)
// 	if err != nil {
// 		return "", err
// 	}

// 	if !user.IsOTPValid {
// 		return "", fmt.Errorf("OTP is not valid for email: %s", email)
// 	}

// 	return user.OTP, nil
// }

// func ClearStoredOTP(email string) error {
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	filter := bson.M{"email": email}
// 	update := bson.M{
// 		"$set": bson.M{
// 			"otp":          "",
// 			"is_otp_valid": false,
// 			"updated_at":   time.Now(),
// 		},
// 	}

// 	_, err := userCollection.UpdateOne(ctx, filter, update)
// 	return err
// }

// package tasks

// import (
// 	"context"
// 	"fmt"
// 	"math/rand"
// 	"time"

// 	"github.com/go-redis/redis/v8"
// )

// var redisClient *redis.Client

// func init() {
// 	redisClient = redis.NewClient(&redis.Options{
// 		Addr: "localhost:6379", // Update this with your Redis server address
// 	})
// }

// func GenerateOTP(email string) (string, error) {
// 	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
// 	return otp, nil
// }

// func StoreOTP(email, otp string) error {
// 	ctx := context.Background()
// 	err := redisClient.Set(ctx, email, otp, 15*time.Minute).Err()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
