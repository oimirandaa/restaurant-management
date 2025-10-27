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

type InvoiceViewFormat struct {
	InvoiceId      string
	PaymentMethod  string
	OrderId        string
	PaymentStatus  *string
	PaymentDue     interface{}
	TableNumber    interface{}
	PaymentDueDate time.Time
	OrderDetails   interface{}
}

var invoiceCollection *mongo.Collection = database.OpenCollection(database.Client, "invoice")

func GetInvoices() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		result, err := invoiceCollection.Find(context.TODO(), bson.M{})
		defer func(result *mongo.Cursor, ctx context.Context) {
			err := result.Close(ctx)
			if err != nil {

			}
		}(result, ctx)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching orders"})
			return
		}

		var allInvoices []bson.M
		if err = result.All(ctx, &allInvoices); err != nil {
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, allInvoices)
	}
}

func GetInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		invoiceId := c.Param("invoice_id")
		var invoice models.Invoice

		err := invoiceCollection.FindOne(ctx, bson.M{"invoice_id": invoiceId}).Decode(&invoice)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching invoice"})
		}
		//c.JSON(http.StatusOK, invoice)

		var invoiceView InvoiceViewFormat

		allOrdersItems, err := ItemsByOrder(invoice.OrderId)

		invoiceView.OrderId = invoice.OrderId
		invoiceView.PaymentDueDate = invoice.PaymentDueDate
		invoiceView.PaymentMethod = "null"

		if invoice.PaymentMethod != nil {
			invoiceView.PaymentMethod = *invoice.PaymentMethod
		}

		invoiceView.InvoiceId = invoice.InvoiceId
		invoiceView.PaymentStatus = *&invoice.PaymentStatus
		invoiceView.PaymentDue = allOrdersItems[0]["payment_due"]
		invoiceView.TableNumber = allOrdersItems[0]["table_number"]
		invoiceView.OrderDetails = allOrdersItems[0]["order_items"]

		c.JSON(http.StatusOK, invoiceView)
	}
}

func CreateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var invoice models.Invoice
		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var order models.Order
		err := orderCollection.FindOne(ctx, bson.M{"order_id": invoice.OrderId}).Decode(&order)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred while fetching order"})
			return
		}

		status := "PENDING"

		if invoice.PaymentStatus == nil {
			invoice.PaymentStatus = &status
		}

		invoice.PaymentDueDate, _ = time.Parse(time.RFC3339, time.Now().AddDate(0, 0, 1).Format(time.RFC3339))
		invoice.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		invoice.ID = primitive.NewObjectID()
		invoice.InvoiceId = invoice.ID.Hex()

		validationErr := validate.Struct(invoice)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			return
		}

		insertResult, insertErr := invoiceCollection.InsertOne(ctx, invoice)
		if insertErr != nil {
			msg := fmt.Sprintf("error ocurred while inserting the invoice %s", insertErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusCreated, insertResult)
	}
}

func UpdateInvoice() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()

		var invoice models.Invoice
		invoiceId := c.Param("invoice_id")

		if err := c.BindJSON(&invoice); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		filter := bson.M{"invoice_id": invoiceId}
		var updateObj primitive.D

		if invoice.PaymentStatus != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_method", Value: invoice.PaymentMethod})
		}

		if invoice.PaymentStatus != nil {
			updateObj = append(updateObj, bson.E{Key: "payment_status", Value: invoice.PaymentStatus})

		}

		invoice.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		updateObj = append(updateObj, bson.E{Key: "updated_at", Value: invoice.UpdatedAt})

		upsert := true
		opt := options.UpdateOptions{
			Upsert: &upsert,
		}

		status := "PENDING"
		if invoice.PaymentStatus == nil {
			invoice.PaymentStatus = &status
		}

		result, err := invoiceCollection.UpdateOne(
			ctx,
			filter,
			bson.D{
				{"$set", updateObj},
			},
			&opt,
		)
		if err != nil {
			msg := fmt.Sprintf("error ocurred while updating the invoice %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
