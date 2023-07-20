package main

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// item represents one purchased item on the receipt with a short description and price
type item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

// receipt represents a purchase receipt containing details of a transaction
type receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []item `json:"items"`
	Total        string `json:"total"`
	ID           string `json:"id"`
	Points       int    `json:"points"`
}

// returnID represents an ID given to a processed receipt
type returnID struct {
	ID string `json:"id"`
}

// returnPoints represents the number of points awarded for a receipt
type returnPoints struct {
	Points int `json:"points"`
}

// receipts is an array containing all currently processed receipts
// array is cleared at the end of each run //RKS not sure if this is necessary
var receipts = []receipt{}

// getReceipts sends a JSON response containing a list of all processed receipts (used for testing)
func getReceipts(context *gin.Context) {
	context.IndentedJSON(http.StatusOK, receipts)
}

// processReceipt takes in a JSON receipt and returns a JSON object containing the generated ID for the receipt.
func processReceipt(context *gin.Context) {
	var newReceipt receipt

	// generate and assign a unique ID to the receipt
	newId := uuid.NewString()
	newReceipt.ID = newId

	// check if new receipt is valid
	if err := context.BindJSON(&newReceipt); err != nil {
		context.IndentedJSON(http.StatusBadRequest, gin.H{"message": "The receipt is invalid"})
		return
	}

	// if valid, add receipt to receipts array and return the assigned ID
	receipts = append(receipts, newReceipt)
	returnID := returnID{
		ID: newId,
	}
	context.IndentedJSON(http.StatusOK, returnID)
}

// getPoints takes in a receipt ID and returns a JSON object containing the points awarded for that receipt
func getPoints(context *gin.Context) {
	// grab id and look for matching receipt
	id := context.Param("id")
	receipt, err := getReceiptById(id)
	if err != nil {
		context.IndentedJSON(http.StatusNotFound, gin.H{"message": "No receipt found for that id"})
		return
	}

	// return point total right away if it has already been calculated
	if receipt.Points != 0 {
		context.IndentedJSON(http.StatusOK, returnPoints{Points: receipt.Points})
		return
	}

	pointTotal := 0 // running tally for receipt points

	// add one point for every alphanumeric char in retailer name
	for _, char := range receipt.Retailer {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			pointTotal++
		}
	}

	// parse receipt total
	totalFloat, err := strconv.ParseFloat(receipt.Total, 64)
	if err != nil {
		context.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Unable to calculate points (invalid total)"})
		return
	}

	// add 50 points if the receipt total is a round dollar amount with no cents
	if math.Mod(totalFloat, 1) == 0 {
		pointTotal += 50
	}

	// add 25 points if the receipt total is a multiple of 0.25.
	if math.Mod(totalFloat, .25) == 0 {
		pointTotal += 25
	}

	// add 5 points for every two items on the receipt.
	pointTotal += (len(receipt.Items) / 2) * 5

	// iterate through every item listed on the receipt
	// if the trimmed length of the item description is a multiple of 3,
	// multiply the price by 0.2 and round up to the nearest integer. Add that many points.
	for _, item := range receipt.Items {
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {

			// parse price of item
			priceFloat, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				context.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Unable to calculate points (invalid item price(s))"})
				return
			}

			pointTotal += int(math.Ceil(priceFloat * .2))
		}
	}

	// add 6 points if the day in the purchase date is odd.
	dayInt, err := strconv.Atoi(receipt.PurchaseDate[8:10]) // parse day of purchase, chars 8&9 in YYYY-MM-DD format
	if err != nil {
		context.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Unable to calculate points (invalid date of purchase)"})
		return
	}

	if dayInt%2 == 1 {
		pointTotal += 6
	}

	// add 10 points if the time of purchase is after 2:00pm (inclusive) and before 4:00pm (exclusive)
	hourInt, err := strconv.Atoi(receipt.PurchaseTime[0:2]) // parse hour of purchase, chars 0&1 in HH:MM format
	if err != nil {
		context.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Unable to calculate points (invalid time of purchase)"})
		return
	}

	if hourInt == 14 || hourInt == 15 {
		pointTotal += 10
	}

	// save point total to receipt struct and return
	receipt.Points = pointTotal
	context.IndentedJSON(http.StatusOK, returnPoints{Points: receipt.Points})
}

// getReceiptById is a helper function that takes in a string id and returns the corresponding receipt
func getReceiptById(id string) (*receipt, error) {
	for i, r := range receipts {
		if r.ID == id {
			return &receipts[i], nil
		}
	}

	// no match found, return error message
	return nil, errors.New("no receipt found for that id")
}

// main is the entry point of the Gin web application.
// It sets up the router, defines the endpoints, and starts the server.
func main() {
	// create a new Gin router
	router := gin.Default()

	// define endpoints and their corresponding handler functions.
	router.GET("/receipts", getReceipts)
	router.POST("/receipts/process", processReceipt)
	router.GET("/receipts/:id/points", getPoints)

	// start the server and listen on localhost:9090
	router.Run("localhost:9090")
}
