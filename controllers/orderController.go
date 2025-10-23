package controllers

import (
	"context"
	"fmt"
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

var orderCollection *mongo.Collection = database.OpenCollection(database.Client, "orders")

func GetOrders() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := orderCollection.Find(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching orders"})
		}
		var allOrders []bson.M
		if err = result.All(ctx, &allOrders); err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, allOrders)
	}
}

func GetOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		orderId := c.Param("order_id")
		var order models.Order

		err := orderCollection.FindOne(ctx, bson.M{"order_id": orderId}).Decode(&order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching order"})
		}
		c.JSON(http.StatusOK, order)
	}
}

func CreateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var table models.Table
		var order models.Order

		defer cancel()

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		validationErr := validate.Struct(order)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		if order.TableId != nil {
			err := tableCollection.FindOne(ctx, bson.M{"table_id": order.TableId}).Decode(&table)
			defer cancel()
			if err != nil {
				msg := fmt.Sprintf("Table was not found with id %s", order.TableId)
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}
		}

		order.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		order.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

		order.ID = primitive.NewObjectID()
		order.OrderId = order.ID.Hex()

		result, err := orderCollection.InsertOne(ctx, order)

		if err != nil {
			msg := fmt.Sprintf("error ocurred while inserting the order %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusCreated, result)
	}
}

func UpdateOrder() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var order models.Order
		var table models.Table
		defer cancel()

		orderId := c.Param("order_id")
		var updateObj primitive.D

		if err := c.BindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if order.TableId != nil {
			err := menuCollection.FindOne(ctx, bson.M{"table_id": order.TableId}).Decode(&table)
			defer cancel()

			if err != nil {
				msg := fmt.Sprintf("Menu was not found with id %s", order.TableId)
				c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
				return
			}

			updateObj = append(updateObj, bson.E{"menu", order.TableId})
		}

		order.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: order.UpdatedAt})

		upsert := true
		filter := bson.M{"order_id": orderId}
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		result, err := orderCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)

		if err != nil {
			msg := fmt.Sprintf("Order Item update failed %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func OrderItemOrderCreator(order models.Order) string {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	order.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	order.ID = primitive.NewObjectID()
	order.OrderId = order.ID.Hex()

	_, err := orderCollection.InsertOne(ctx, order)
	if err != nil {
		return "error occurred while inserting the order"
	}

	return order.OrderId
}
