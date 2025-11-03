package controllers

import (
	"context"
	"log"
	"net/http"
	"restaurant-management-system/database"
	"restaurant-management-system/models"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderItemPack struct {
	TableId    *string
	OrderItems []models.OrderItem
}

var orderItemsCollection *mongo.Collection = database.OpenCollection(database.Client, "orderItem")

func GetOrderItems() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := orderItemsCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing ordered items"})
			return
		}

		var allOrderedItems []bson.M
		if err = result.All(ctx, &allOrderedItems); err != nil {
			log.Fatal(err)
			return
		}

		c.JSON(http.StatusOK, allOrderedItems)
	}
}

func GetOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("order_id")
		allOrderItems, err := ItemsByOrder(orderId)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching order items"})
			return
		}
		c.JSON(http.StatusOK, allOrderItems)
	}
}

func GetOrderItemsByOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		orderItemId := c.Param("order_item_id")
		var orderItem models.OrderItem

		err := orderItemsCollection.FindOne(ctx, bson.M{"order_item_id": orderItemId}).Decode(&orderItem)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching order items"})
			return
		}
		c.JSON(http.StatusOK, orderItem)
	}
}

func CreateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var orderItemPack OrderItemPack
		var order models.Order
		defer cancel()

		if err := c.BindJSON(&orderItemPack); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order.OrderDate, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		orderItemsToBeInserted := []interface{}{}
		order.TableId = orderItemPack.TableId
		order_id := OrderItemOrderCreator(order)

		for _, orderItem := range orderItemPack.OrderItems {
			orderItem.OrderId = order_id

			validationErr := validate.Struct(orderItem)

			if validationErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
				return
			}

			orderItem.ID = primitive.NewObjectID()
			orderItem.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
			orderItem.OrderItemId = orderItem.ID.Hex()

			var number = toFixed(*orderItem.UnitPrice, 2)
			orderItem.UnitPrice = &number

			orderItemsToBeInserted = append(orderItemsToBeInserted, orderItem)
		}

		insertResult, insertErr := orderItemsCollection.InsertMany(ctx, orderItemsToBeInserted)
		if insertErr != nil {
			log.Fatal(insertErr)
			return
		}

		c.JSON(http.StatusCreated, insertResult)
	}
}

func UpdateOrderItem() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var orderItem models.OrderItem
		defer cancel()

		orderItemId := c.Param("order_item_id")
		filter := bson.M{"order_item_id": orderItemId}

		var updateObj primitive.D

		if orderItem.UnitPrice != nil {
			updateObj = append(updateObj, bson.E{Key: "unit_price", Value: *&orderItem.UnitPrice})
		}
		if orderItem.Quantity != nil {
			updateObj = append(updateObj, bson.E{Key: "quantity", Value: *orderItem.Quantity})
		}
		if orderItem.FoodId != nil {
			updateObj = append(updateObj, bson.E{Key: "food_id", Value: *orderItem.FoodId})
		}

		orderItem.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{"updated_at", orderItem.UpdatedAt})

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, errResult := orderItemsCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{
					"$set", updateObj,
				},
			},
			&opt,
		)

		if errResult != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred while updating order item"})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func ItemsByOrder(orderId string) (OrderItems []primitive.M, err error) {
	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	matchStage := bson.D{
		{"$match", bson.D{
			{"order_id", orderId},
		}},
	}

	lookupFoodStage := bson.D{
		{
			"$lookup", bson.D{
				{"from", "food"},
				{"localField", "food_id"},
				{"foreignField", "food_id"},
				{"as", "food"},
			},
		},
	}
	unwindFoodStage := bson.D{
		{
			"$unwind", bson.D{
				{"path", "$food"},
				{"preserveNullAndEmptyArrays", true}},
		},
	}

	lookupOrderStage := bson.D{
		{
			"$lookup", bson.D{
				{"from", "order"},
				{"localField", "order_id"},
				{"foreignField", "order_id"},
				{"as", "order"},
			},
		},
	}
	unwindOrderStage := bson.D{
		{
			"$unwind", bson.D{
				{"path", "$order"},
				{"preserveNullAndEmptyArrays", true}},
		},
	}

	lookupTableStage := bson.D{
		{
			"$lookup", bson.D{
				{"from", "table"},
				{"localField", "order.table_id"},
				{"foreignField", "table_id"},
				{"as", "table"},
			},
		},
	}
	unwindTableStage := bson.D{
		{
			"$unwind", bson.D{
				{"path", "$table"},
				{"preserveNullAndEmptyArrays", true}},
		},
	}

	projectStage := bson.D{
		{
			"$project", bson.D{
				{"_id", 0},
				{"amount", "$food.price"},
				{"total_count", 1},
				{"food_name", "$food.name"},
				{"food_image", "$food.food_image"},
				{"table_number", "$table.table_number"},
				{"table_id", "$table.table_id"},
				{"order_id", "$order.order_id"},
				{"price", "$food.price"},
				{"quantity", 1},
			}},
	}

	groupStage := bson.D{
		{
			"$group", bson.D{
				{
					"_id", bson.D{
						{"order_id", "$order_id"},
						{"table_id", "$table_id"},
						{"table_number", "$table_number"},
					},
				},
				{
					"payment_due", bson.D{
						{"$sum", "$amount"},
					},
				},
				{
					"total_count", bson.D{
						{"$sum", 1},
					},
				},
				{
					"order_items", bson.D{
						{"$push", "$order_items"},
					},
				},
			},
		},
	}

	projectStage2 := bson.D{
		{
			"$project", bson.D{
				{"_id", 0},
				{"payment_due", 1},
				{"total_count", 1},
				{"table_number", "$_id.table_number"},
				{"order_items", 1},
				{"table_id", 1},
				{"order_id", 1},
			},
		},
	}

	result, err := orderItemsCollection.Aggregate(
		ctx,
		mongo.Pipeline{
			matchStage,
			lookupFoodStage,
			unwindFoodStage,
			lookupOrderStage,
			unwindOrderStage,
			lookupTableStage,
			unwindTableStage,
			projectStage,
			groupStage,
			projectStage2,
		},
	)

	if err != nil {
		panic(err)
		return nil, err
	}

	if err = result.All(ctx, &OrderItems); err != nil {
		panic(err)
	}

	return OrderItems, nil
}
