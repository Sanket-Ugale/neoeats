package controller

import (
	"context"
	"fmt"
	"golang-restaurant-management/database"
	"golang-restaurant-management/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "order")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderCollection.Find(context.TODO(), bson.M{})
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing order items"})
		}

		var allOrders []bson.M
		if err = result.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}
		if allOrders == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No orders found"})
		} else {
			c.JSON(http.StatusOK, allOrders)
		}
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		orderId := c.Param("order_id")
		var order models.Order

		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while fetching the orders"})
		}
		c.JSON(http.StatusOK, order)
	}
}

// func CreateOrder() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var table models.Table
// 		var order models.Order

// 		if err := c.BindJSON(&order); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		validationErr := validate.Struct(order)

// 		if validationErr != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
// 			return
// 		}

// 		if order.Table_id != nil {
// 			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.Table_id}).Decode(&table)
// 			defer cancel()
// 			if err != nil {
// 				msg := fmt.Sprintf("message:Table was not found")
// 				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
// 				return
// 			}
// 		}

// 		order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
// 		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

// 		order.ID = primitive.NewObjectID()
// 		order.Order_id = order.ID.Hex()

// 		result, insertErr := orderCollection.InsertOne(ctx, order)

// 		if insertErr != nil {
// 			msg := fmt.Sprintf("order item was not created")
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
// 			return
// 		}

// 		defer cancel()
// 		c.JSON(http.StatusOK, result)
// 	}
// }

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var order models.Order

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(order)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		if order.Table_id != nil {
			var table models.Table
			err := tableCollection.FindOne(ctx, bson.M{"table_id": *order.Table_id}).Decode(&table)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while finding table"})
				}
				return
			}
		}

		order.Created_at = time.Now()
		order.Updated_at = time.Now()
		order.ID = primitive.NewObjectID()
		order.Order_id = order.ID.Hex()

		result, insertErr := orderCollection.InsertOne(ctx, order)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order was not created"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "Order created successfully",
			"order_id":    order.Order_id,
			"inserted_id": result.InsertedID,
		})
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var updateData struct {
			Order_Date time.Time `json:"order_date"`
			Table_id   string    `json:"table_id"`
		}

		orderId := c.Param("order_id")

		if err := c.BindJSON(&updateData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var updateObj primitive.D

		if !updateData.Order_Date.IsZero() {
			updateObj = append(updateObj, bson.E{"order_date", updateData.Order_Date})
		}

		if updateData.Table_id != "" {
			// Optionally, you can check if the table exists
			var table models.Table
			err := tableCollection.FindOne(ctx, bson.M{"table_id": updateData.Table_id}).Decode(&table)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					c.JSON(http.StatusNotFound, gin.H{"error": "Table not found"})
				} else {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while finding table"})
				}
				return
			}
			updateObj = append(updateObj, bson.E{"table_id", updateData.Table_id})
		}

		updateObj = append(updateObj, bson.E{"updated_at", time.Now()})

		filter := bson.M{"order_id": orderId}

		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{{"$set", updateObj}},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Order update failed"})
			return
		}

		if result.MatchedCount == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Order updated successfully",
			"modified_count": result.ModifiedCount,
		})
	}
}

// func UpdateOrder() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var table models.Table
// 		var order models.Order

// 		var updateObj primitive.D

// 		orderId := c.Param("order_id")
// 		if err := c.BindJSON(&order); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		if order.Table_id != nil {
// 			err := orderCollection.FindOne(ctx, bson.M{"order_id": order.Order_id}).Decode(&order)
// 			defer cancel()
// 			if err != nil {
// 				msg := fmt.Sprintf("message:order was not found")
// 				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
// 				return
// 			}
// 			// updateObj = append(updateObj, bson.E{"menu", order.Table_id})
// 		}

// 		order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
// 		updateObj = append(updateObj, bson.E{"updated_at", time.Now() })

// 		upsert := true

// 		filter := bson.M{"order_id": orderId}
// 		opt := options.UpdateOptions{
// 			Upsert: &upsert,
// 		}

// 		result, err := orderCollection.UpdateOne(
// 			ctx,
// 			filter,
// 			bson.D{
// 				{"$st", updateObj},
// 			},
// 			&opt,
// 		)

// 		if err != nil {
// 			msg := fmt.Sprintf("order item update failed")
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
// 			return
// 		}

// 		defer cancel()
// 		c.JSON(http.StatusOK, result)
// 	}
// }

func OrderItemOrderCreator(order models.Order) string {

	order.Created_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.ID = primitive.NewObjectID()
	order.Order_id = order.ID.Hex()

	orderCollection.InsertOne(ctx, order)
	defer cancel()

	return order.Order_id
}

func DeleteOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		orderId := c.Param("order_id")

		result, err := orderCollection.DeleteOne(ctx, bson.M{"order_id": orderId})
		defer cancel()

		if result.DeletedCount == 0 {
			msg := fmt.Sprintf("order item not found")
			c.JSON(http.StatusNotFound, gin.H{"error": msg})
			return
		}

		if err != nil {
			msg := fmt.Sprintf("order item was not deleted")
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order item deleted", "DeletedCount": result.DeletedCount})
	}
}
