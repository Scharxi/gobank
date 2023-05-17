package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

func WriteJson(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type apiFunc func(w http.ResponseWriter, r *http.Request) error

func makeHttpHandlerFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			_ = WriteJson(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func getId(r *http.Request) (id int, err error) {
	id, err = strconv.Atoi(mux.Vars(r)["id"])
	return id, err
}

type ApiError struct {
	Error string
}

type ApiServer struct {
	listenerAddr string
	accounts     Storage[Account]
}

func NewApiServer(addr string, accounts Storage[Account]) *ApiServer {
	return &ApiServer{
		listenerAddr: addr,
		accounts:     accounts,
	}
}

func (s *ApiServer) Run() {
	router := mux.NewRouter()

	router.HandleFunc("/account/transactions/{id}/details", makeHttpHandlerFunc(s.handleTransactions))
	router.HandleFunc("/account/transactions/{id}", makeHttpHandlerFunc(s.handleGetTransactions))
	router.HandleFunc("/account/{id}", makeHttpHandlerFunc(s.handleDeleteAccount))
	router.HandleFunc("/account", makeHttpHandlerFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", makeHttpHandlerFunc(s.handleGetAccountById))

	log.Println("JSON API server running on PORT: ", s.listenerAddr)

	if err := http.ListenAndServe(s.listenerAddr, router); err != nil {
		log.Fatalln(err)
	}
}

func (s *ApiServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}
	if r.Method == "POST" {
		return s.handleCreateAccount(w, r)
	}
	/*if r.Method == "DELETE" {
		return s.handleDeleteAccount(w, r)
	}*/
	if r.Method == "PUT" {
		return s.handleTransfer(w, r)
	}
	return fmt.Errorf("method `%s` is not allowed", r.Method)
}

func (s *ApiServer) handleGetAccountById(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	log.Println("Getting account with id: ", id)
	acc, err := s.accounts.GetById(id)
	if err != nil {
		return err
	}

	return WriteJson(w, http.StatusOK, acc)
}

func (s *ApiServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.accounts.Get()
	if err != nil {
		return err
	}
	fmt.Println("Getting Accounts")
	return WriteJson(w, http.StatusOK, accounts)
}

func (s *ApiServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountRequest := new(CreateAccountRequest)

	if !createAccountRequest.Validate() {
		return fmt.Errorf("invalid input")
	}

	if err := json.NewDecoder(r.Body).Decode(createAccountRequest); err != nil {
		return err
	}

	// save the account in the db
	account := NewAccount(createAccountRequest.FirstName, createAccountRequest.LastName)
	if err := s.accounts.Create(account); err != nil {
		return err
	}
	return WriteJson(w, http.StatusCreated, createAccountRequest)
}

func (s *ApiServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}

	err = s.accounts.Delete(id)
	if err != nil {
		return err
	}

	return WriteJson(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("Successfully deleted account with id `%d`", id)})
}

func (s *ApiServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	makeTransactionRequest := new(MakeTransactionRequest)

	if err := json.NewDecoder(r.Body).Decode(makeTransactionRequest); err != nil {
		return err
	}

	// save the account in the db
	transaction := NewTransaction(makeTransactionRequest.AccountId, makeTransactionRequest.RecipientId, makeTransactionRequest.Amount)
	storage := s.accounts.(*PostgresStorage)
	if err := storage.CreateTransaction(transaction); err != nil {
		return err
	}
	return WriteJson(w, http.StatusOK, makeTransactionRequest)
}

func (s *ApiServer) handleGetTransactions(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	accWithTransactions, err := s.accounts.(*PostgresStorage).GetAccountWithTransactions(id)
	if err != nil {
		return err
	}

	return WriteJson(w, http.StatusOK, accWithTransactions)
}

func (s *ApiServer) handleTransactions(w http.ResponseWriter, r *http.Request) error {
	if r.Method == http.MethodPost {
		return s.handleAddTransactionDetails(w, r)
	}
	if r.Method == http.MethodGet {
		return s.handleGetTransactionDetails(w, r)
	}
	if r.Method == http.MethodPut {
		return s.handleUpdateTransactionDetails(w, r)
	}
	return nil
}

func (s *ApiServer) handleAddTransactionDetails(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	detailsRequest := new(TransactionDetailsRequest)
	if err := json.NewDecoder(r.Body).Decode(detailsRequest); err != nil {
		return err
	}

	details := NewTransactionDetails(id, detailsRequest.Description, detailsRequest.Tags)
	err = s.accounts.(*PostgresStorage).CreateTransactionDetails(details)
	if err != nil {
		return err
	}
	fmt.Println("Test2")
	return WriteJson(w, http.StatusCreated, detailsRequest)
}

func (s *ApiServer) handleGetTransactionDetails(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	tWithDetails, err := s.accounts.(*PostgresStorage).GetTransactionWithDetails(id)
	if err != nil {
		return err
	}
	return WriteJson(w, http.StatusOK, tWithDetails)
}

func (s *ApiServer) handleUpdateTransactionDetails(w http.ResponseWriter, r *http.Request) error {
	id, err := getId(r)
	if err != nil {
		return err
	}
	update := new(TransactionDetailsRequest)

	if err := json.NewDecoder(r.Body).Decode(update); err != nil {
		return err
	}

	err = s.accounts.(*PostgresStorage).UpdateTransactionDetails(id, update)
	if err != nil {
		return err
	}
	return WriteJson(w, http.StatusOK, update)
}
