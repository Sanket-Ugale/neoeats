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
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderItemPack struct {
	Table_id    *string
	Order_items []models.OrderItem
}

var orderItemCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		result, err := orderItemCollection.Find(context.TODO(), bson.M{})
		defer cancel()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing ordered items"})
			return
		}
		var allOrderItems []bson.M
		if err = result.All(ctx, &allOrderItems); err != nil {
			log.Fatal(err)
			return
		}
		if allOrderItems == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No order items found"})

		} else {
			c.JSON(http.StatusOK, allOrderItems)
		}

	}
}
func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")

		allOrderItems, err := ItemsByOrder(orderId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error occurred while listing order items by order ID: %s", err.Error())})
			return
		}
		c.JSON(http.StatusOK, allOrderItems)
	}
}

func ItemsByOrder(id string) (OrderItems []primitive.M, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	matchStage := bson.D{{"$match", bson.D{{"order_id", id}}}}
	lookupStage := bson.D{{"$lookup", bson.D{{"from", "food"}, {"localField", "food_id"}, {"foreignField", "food_id"}, {"as", "food"}}}}
	unwindStage := bson.D{{"$unwind", bson.D{{"path", "$food"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupOrderStage := bson.D{{"$lookup", bson.D{{"from", "order"}, {"localField", "order_id"}, {"foreignField", "order_id"}, {"as", "order"}}}}
	unwindOrderStage := bson.D{{"$unwind", bson.D{{"path", "$order"}, {"preserveNullAndEmptyArrays", true}}}}

	lookupTableStage := bson.D{{"$lookup", bson.D{{"from", "table"}, {"localField", "order.table_id"}, {"foreignField", "table_id"}, {"as", "table"}}}}
	unwindTableStage := bson.D{{"$unwind", bson.D{{"path", "$table"}, {"preserveNullAndEmptyArrays", true}}}}

	projectStage := bson.D{
		{"$project", bson.D{
			{"amount", "$food.price"},
			{"total_count", 1},
			{"food_name", "$food.name"},
			{"food_image", "$food.food_image"},
			{"table_number", "$table.table_number"},
			{"table_id", "$table.table_id"},
			{"order_id", "$order.order_id"},
			{"price", "$food.price"},
			{"quantity", 1},
		}}}

	groupStage := bson.D{{"$group", bson.D{
		{"_id", bson.D{
			{"order_id", "$order_id"},
			{"table_id", "$table_id"},
			{"table_number", "$table_number"},
		}},
		{"payment_due", bson.D{{"$sum", "$amount"}}},
		{"total_count", bson.D{{"$sum", 1}}},
		{"order_items", bson.D{{"$push", "$$ROOT"}}},
	}}}

	projectStage2 := bson.D{
		{"$project", bson.D{
			{"payment_due", 1},
			{"total_count", 1},
			{"table_number", "$_id.table_number"},
			{"order_items", 1},
		}}}

	result, err := orderItemCollection.Aggregate(ctx, mongo.Pipeline{
		matchStage,
		lookupStage,
		unwindStage,
		lookupOrderStage,
		unwindOrderStage,
		lookupTableStage,
		unwindTableStage,
		projectStage,
		groupStage,
		projectStage2})

	if err != nil {
		return nil, fmt.Errorf("error in aggregation: %w", err)
	}

	if err = result.All(ctx, &OrderItems); err != nil {
		return nil, fmt.Errorf("error decoding result: %w", err)
	}

	return OrderItems, nil
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderItemId := c.Param("order_item_id")

		var orderItem models.OrderItem

		objId, err := primitive.ObjectIDFromHex(orderItemId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order item ID format"})
			return
		}

		err = orderItemCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&orderItem)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": "No order item found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching ordered item"})
			}
			return
		}

		c.JSON(http.StatusOK, orderItem)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		var orderItem models.OrderItem

		orderItemId := c.Param("order_item_id")

		filter := bson.M{"order_item_id": orderItemId}

		var updateObj primitive.D

		if orderItem.Unit_price != nil {
			updateObj = append(updateObj, bson.E{"unit_price", *&orderItem.Unit_price})
		}

		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{"quantity", *orderItem.Quantity})
		}

		if orderItem.Food_id != nil {
			updateObj = append(updateObj, bson.E{"food_id", *orderItem.Food_id})
		}

		orderItem.Updated_at, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.Updated_at})

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderItemCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)

		if err != nil {
			msg := "Order item update failed"
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var orderItemPack OrderItemPack
		var order models.Order

		if err := c.BindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if orderItemPack.Table_id == nil || len(*orderItemPack.Table_id) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Table ID is required"})
			return
		}

		if orderItemPack.Order_items == nil || len(orderItemPack.Order_items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Order items are required"})
			return
		}

		order.Order_Date, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.Table_id = orderItemPack.Table_id
		order_id := OrderItemOrderCreator(order)

		orderItemsToBeInserted := []interface{}{}

		for _, orderItem := range orderItemPack.Order_items {
			orderItem.Order_id = order_id

			if orderItem.Quantity == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity is required"})
				return
			}

			if orderItem.Food_id == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Food ID is required"})
				return
			}

			// Fetch the food item from the database using Food_id
			var food models.Food
			err := foodCollection.FindOne(ctx, bson.M{"food_id": *orderItem.Food_id}).Decode(&food)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Food item not found"})
				return
			}

			orderItem.Unit_price = food.Price
			totalPrice := *orderItem.Unit_price * float64(*orderItem.Quantity)
			orderItem.Total_price = &totalPrice

			orderItem.ID = primitive.NewObjectID()
			orderItem.Created_at = time.Now()
			orderItem.Updated_at = time.Now()
			orderItem.Order_item_id = orderItem.ID.Hex()

			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		insertedOrderItems, err := orderItemCollection.InsertMany(ctx, orderItemsToBeInserted)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, insertedOrderItems)
	}
}

func DeleteOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)

		orderItemId := c.Param("order_item_id")

		result, err := orderItemCollection.DeleteOne(ctx, bson.M{"order_item_id": orderItemId})

		if result.DeletedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Not found order item"})
			return
		}

		if err != nil {
			msg := "Order item delete failed"
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		defer cancel()

		c.JSON(http.StatusOK, result)
	}
}
