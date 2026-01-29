// Package models implements a way to store orders in memory
package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// An Order is a structure to keep an order with its payment, delivery and items
type Order struct {
	OrderUID          string    `json:"order_uid" db:"order_uid"`
	TrackNumber       string    `json:"track_number" db:"track_number"`
	Entry             string    `json:"entry" db:"entry"`
	DeliveryID        int64     `db:"delivery_id"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items"`
	Locale            string    `json:"locale" db:"locale"`
	InternalSignature string    `json:"internal_signature" db:"internal_signature"`
	CustomerID        string    `json:"customer_id" db:"customer_id"`
	DeliveryService   string    `json:"delivery_service" db:"delivery_service"`
	Shardkey          string    `json:"shardkey" db:"shardkey"`
	SmID              int       `json:"sm_id" db:"sm_id"`
	DateCreated       time.Time `json:"date_created" db:"date_created"`
	OofShard          string    `json:"oof_shard" db:"oof_shard"`
}

// A Delivery is a structure to keep information about order delivery
type Delivery struct {
	Name    string `json:"name" db:"name"`
	Phone   string `json:"phone" db:"phne"`
	Zip     string `json:"zip" db:"zip"`
	City    string `json:"city" db:"city"`
	Address string `json:"address" db:"address"`
	Region  string `json:"region" db:"region"`
	Email   string `json:"email" db:"email"`
}

// A Payment is a structure to keep information about order payment
type Payment struct {
	Transaction  string `json:"transaction" db:"transaction"`
	RequestID    string `json:"request_id" db:"request_id"`
	Currency     string `json:"currency" db:"currency"`
	Provider     string `json:"provider" db:"provider"`
	Amount       int    `json:"amount" db:"amount"`
	PaymentDt    int64  `json:"payment_dt" db:"payment_dt"`
	Bank         string `json:"bank" db:"bank"`
	DeliveryCost int    `json:"delivery_cost" db:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total" db:"goods_total"`
	CustomFee    int    `json:"custom_fee" db:"custom_fee"`
}

// An Item is a structure to keep information about one order item
type Item struct {
	ChrtID      int64  `json:"chrt_id" db:"chrt_id"`
	TrackNumber string `json:"track_number" db:"track_number"`
	Price       int    `json:"price" db:"price"`
	Rid         string `json:"rid" db:"rid"`
	Name        string `json:"name" db:"name"`
	Sale        int    `json:"sale" db:"sale"`
	Size        string `json:"size" db:"size"`
	TotalPrice  int    `json:"total_price" db:"total_price"`
	NmID        int64  `json:"nm_id" db:"nm_id"`
	Brand       string `json:"brand" db:"brand"`
	Status      int    `json:"status" db:"status"`
}

// A ValidationError is a custom error type for data validation
type ValidationError struct {
	Field   string
	Struct  string
	Message string
}

// Error is an interface implementation for errors
func (e ValidationError) Error() string {
	return fmt.Sprintf("Validation error in field %s.%s: %s", e.Struct, e.Field, e.Message)
}

// NewOrderValidationError is a validation error in the Order
func NewOrderValidationError(field, message string) ValidationError {
	return ValidationError{field, "order", message}
}

// NewDeliveryValidationError is a validation error in the Delivery
func NewDeliveryValidationError(field, message string) ValidationError {
	return ValidationError{field, "delivery", message}
}

// NewPaymentValidationError is a validation error in the Payment
func NewPaymentValidationError(field, message string) ValidationError {
	return ValidationError{field, "payment", message}
}

// NewItemValidationError is a validation error in the Item
func NewItemValidationError(field, message string) ValidationError {
	return ValidationError{field, "item", message}
}

// Validate checks if the Order data is correct
func (o *Order) Validate() error {
	if err := o.validateRequired(); err != nil {
		return err
	}
	if err := o.validateLogic(); err != nil {
		return err
	}

	if err := o.Delivery.Validate(); err != nil {
		return err
	}
	if err := o.Payment.Validate(); err != nil {
		return err
	}
	for i, item := range o.Items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("item[%d]:	%w", i, err)
		}
	}

	return nil
}

// validateRequired checks if the required fields of an Order are set
func (o *Order) validateRequired() error {
	if strings.TrimSpace(o.OrderUID) == "" {
		return NewOrderValidationError("order_uid", "is required")
	}

	if strings.TrimSpace(o.TrackNumber) == "" {
		return NewOrderValidationError("track_number", "is required")
	}

	if strings.TrimSpace(o.Entry) == "" {
		return NewOrderValidationError("entry", "is required")
	}

	if strings.TrimSpace(o.CustomerID) == "" {
		return NewOrderValidationError("customer_id", "is required")
	}

	if len(o.Items) == 0 {
		return NewOrderValidationError("items", "at least one item has to be present")
	}

	return nil
}

// validateLogic checks that  values for Order fields are valid
func (o *Order) validateLogic() error {
	orderUIDPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !orderUIDPattern.MatchString(o.OrderUID) {
		return NewOrderValidationError("order_uid", "must contain only letters, digits, underscores and digits")
	}

	if o.DateCreated.After(time.Now()) {
		return NewOrderValidationError("date_created", "cannot be in the future")
	}

	if o.Locale != "" {
		localePattern := regexp.MustCompile(`^[a-z]{2}$`)
		if !localePattern.MatchString(o.Locale) {
			return NewOrderValidationError("locale", "must be a 2-letter language code")
		}
	}

	if o.SmID <= 0 {
		return NewDeliveryValidationError("sm_id", "must be positive")
	}

	itemTotal := 0
	for _, item := range o.Items {
		itemTotal += item.TotalPrice
	}

	if o.Payment.GoodsTotal != itemTotal {
		return NewPaymentValidationError(
			"payment.goods_total",
			fmt.Sprintf("goods_total %d doesnn't match sum of item prices %d", o.Payment.GoodsTotal, itemTotal),
		)
	}

	totalAmount := itemTotal + o.Payment.DeliveryCost + o.Payment.CustomFee
	if o.Payment.Amount != totalAmount {
		return NewOrderValidationError(
			"payment.amount",
			fmt.Sprintf("payment amount %d doesn't match calculated total amount %d", o.Payment.Amount, totalAmount),
		)
	}

	return nil
}

// Validate checks if the Delivery data is correct
func (d *Delivery) Validate() error {
	if err := d.validateRequired(); err != nil {
		return err
	}
	if err := d.validateLogic(); err != nil {
		return err
	}

	return nil
}

// validateRequired checks if the required fields of a Delivery are set
func (d *Delivery) validateRequired() error {
	if strings.TrimSpace(d.Name) == "" {
		return NewDeliveryValidationError("name", "is required")
	}
	if strings.TrimSpace(d.Phone) == "" {
		return NewDeliveryValidationError("name", "is required")
	}
	if strings.TrimSpace(d.Address) == "" {
		return NewDeliveryValidationError("name", "is required")
	}
	if strings.TrimSpace(d.City) == "" {
		return NewDeliveryValidationError("name", "is required")
	}

	return nil
}

// validateLogic checks that  values for Delivery fields are valid
func (d *Delivery) validateLogic() error {
	phonePattern := regexp.MustCompile(`^[\d\s\-+()]+$`)
	if !phonePattern.MatchString(d.Phone) {
		return NewDeliveryValidationError("phone", fmt.Sprintf("invalid phone number: %s", d.Phone))
	}

	if d.Email != "" {
		// emailPattern := regexp.MustCompile(
		// 	`(?:[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\])1(?:[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_` + "`" + `{|}~-]+)*|"(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21\x23-\x5b\x5d-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])*")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\x01-\x08\x0b\x0c\x0e-\x1f\x21-\x5a\x53-\x7f]|\\[\x01-\x09\x0b\x0c\x0e-\x7f])+)\])`,
		// )
		emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailPattern.MatchString(d.Email) {
			return NewDeliveryValidationError("email", fmt.Sprintf("invalid email: %s", d.Email))
		}
	}

	if d.Zip != "" {
		zipPattern := regexp.MustCompile(`^[a-zA-Z0-9\s-]+$`)
		if !zipPattern.MatchString(d.Zip) {
			return NewDeliveryValidationError("zip", fmt.Sprintf("invalid zip code format : %s", d.Zip))
		}
	}

	return nil
}

// Validate checks if the Payment data is correct
func (p *Payment) Validate() error {
	if err := p.validateRequired(); err != nil {
		return err
	}
	if err := p.validateLogic(); err != nil {
		return err
	}

	return nil
}

// validateRequired checks if the required fields of a Payment are set
func (p *Payment) validateRequired() error {
	if strings.TrimSpace(p.Transaction) == "" {
		return NewPaymentValidationError("transaction", "is required")
	}
	if strings.TrimSpace(p.Currency) == "" {
		return NewPaymentValidationError("currency", "is required")
	}
	if strings.TrimSpace(p.Provider) == "" {
		return NewPaymentValidationError("provider", "is required")
	}

	return nil
}

// validateLogic checks that  values for Payment fields are valid
func (p *Payment) validateLogic() error {
	currencyPattern := regexp.MustCompile(`^[A-Z]{3}$`)
	if !currencyPattern.MatchString(p.Currency) {
		return NewPaymentValidationError("currency", "must be a 3-letter currency code")
	}
	if p.Amount < 0 {
		return NewPaymentValidationError("amount", "cannot be negative")
	}
	if p.DeliveryCost < 0 {
		return NewPaymentValidationError("delivery_cost", "cannot be negative")
	}
	if p.GoodsTotal < 0 {
		return NewPaymentValidationError("goods_total", "cannot be negative")
	}
	if p.CustomFee < 0 {
		return NewPaymentValidationError("customm_fee", "cannot be negative")
	}

	if p.PaymentDt > 0 {
		paymentTime := time.Unix(p.PaymentDt, 0)
		if paymentTime.After(time.Now()) {
			return NewPaymentValidationError("payment_dt", "payment date cannot be in future")
		}
	}

	return nil
}

// Validate checks if the Item data is correct
func (i *Item) Validate() error {
	if err := i.validateRequired(); err != nil {
		return err
	}
	if err := i.validateLogic(); err != nil {
		return err
	}

	return nil
}

// validateRequired checks if the required fields of an Item are set
func (i *Item) validateRequired() error {
	if strings.TrimSpace(i.TrackNumber) == "" {
		return NewItemValidationError("track_number", "is required")
	}
	if strings.TrimSpace(i.Name) == "" {
		return NewItemValidationError("name", "is required")
	}
	if strings.TrimSpace(i.Brand) == "" {
		return NewItemValidationError("brand", "is required")
	}

	return nil
}

// validateLogic checks that  values for Item fields are valid
func (i *Item) validateLogic() error {
	if i.ChrtID <= 0 {
		return NewItemValidationError("chrt_id", "must be positive")
	}
	if i.NmID <= 0 {
		return NewItemValidationError("nm_id", "must be positive")

	}
	if i.Price < 0 {
		return NewItemValidationError("price", "cannot be negative")

	}
	if i.TotalPrice < 0 {
		return NewItemValidationError("total_price", "cannot be negative")
	}
	if i.Sale < 0 {
		return NewItemValidationError("sale", "cannot be negative")

	}
	if i.Sale > 100 {
		return NewItemValidationError("sale", "sale percentage cannot be more than 100%")
	}
	expectedPrice := i.Price - (i.Price * i.Sale / 100)
	if i.TotalPrice != expectedPrice {
		return NewItemValidationError(
			"total_price",
			fmt.Sprintf("total price %d doesn't match price %d with sale %d %", i.TotalPrice, i.Price, i.Sale),
		)
	}

	return nil
}