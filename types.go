package main

import (
	"math/rand"
	"time"
)

type Account struct {
	ID           int           `json:"id,omitempty"`
	FirstName    string        `json:"first_name,omitempty"`
	LastName     string        `json:"last_name,omitempty"`
	Number       int64         `json:"account_number,omitempty"`
	Balance      int64         `json:"balance,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	CreateAt     *time.Time    `json:"create_at,omitempty"`
}

type Transaction struct {
	ID              int                  `json:"id"`
	AccountId       int                  `json:"account_id"`
	RecipientId     int                  `json:"recipient_id"`
	Amount          int64                `json:"amount"`
	Details         []TransactionDetails `json:"details,omitempty"`
	TransactionDate time.Time            `json:"transaction_date"`
}

type TransactionDetails struct {
	ID            int      `json:"id,omitempty"`
	TransactionId int      `json:"transaction_id,omitempty"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

func NewTransaction(accountId, recipientId int, amount int64) *Transaction {
	return &Transaction{
		AccountId:       accountId,
		RecipientId:     recipientId,
		Amount:          amount,
		TransactionDate: time.Now().UTC(),
	}
}

func NewAccount(firstName, lastName string) *Account {
	createdAt := time.Now().UTC()
	return &Account{
		FirstName:    firstName,
		LastName:     lastName,
		Number:       int64(rand.Intn(1000000)),
		Balance:      int64(0),
		Transactions: []Transaction{},
		CreateAt:     &createdAt,
	}
}

func NewTransactionDetails(transactionId int, description string, tags []string) *TransactionDetails {
	return &TransactionDetails{
		TransactionId: transactionId,
		Description:   description,
		Tags:          tags,
	}
}

type Validator interface {
	Validate() bool
}

type CreateAccountRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type MakeTransactionRequest struct {
	AccountId   int   `json:"account_id,omitempty"`
	RecipientId int   `json:"recipient_id,omitempty"`
	Amount      int64 `json:"amount,omitempty"`
}

type TransactionDetailsRequest struct {
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (req *CreateAccountRequest) Validate() bool {
	return req.FirstName != "" || req.LastName != ""
}
