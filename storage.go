package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"strings"
	"time"
)

type Storage[T any] interface {
	Get() ([]*T, error)
	Create(*T) error
	Delete(int) error
	Update(*T) error
	GetById(int) (*T, error)
}

type PostgresStorage struct {
	db *sql.DB
}

func (s *PostgresStorage) Init() error {
	if err := s.createTransactionTable(); err != nil {
		return err
	}
	if err := s.createAccountTable(); err != nil {
		return err
	}
	if err := s.createTransactionDetailsTable(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) createAccountTable() error {
	query := `CREATE TABLE if not exists accounts (
   	 	id serial primary key,
	    fist_name varchar(50),
        last_name varchar(50), 
        number serial, 
        balance decimal,
        create_at timestamp
    )`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStorage) createTransactionTable() error {
	query := `CREATE TABLE if not exists transactions (
    	id serial primary key, 
    	account_id serial not null, 
    	recipient_id serial not null,
    	amount decimal,  
    	transaction_date timestamp
    );`

	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStorage) createTransactionDetailsTable() error {
	query := `create table if not exists transaction_details (
		id serial primary key, 
		transaction_id serial not null, 
		description text, 
		tags text
    )`
	_, err := s.db.Exec(query)
	return err
}

func (s PostgresStorage) Get() ([]*Account, error) {
	query := `select * from accounts`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	var accounts []*Account
	for rows.Next() {
		account := new(Account)

		if err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreateAt); err != nil {
			return nil, err
		}

		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (s PostgresStorage) Create(t *Account) error {
	query := `insert into accounts (fist_name, last_name, number, balance, create_at) values ($1, $2, $3, $4, $5)`
	res, err := s.db.Query(
		query,
		t.FirstName,
		t.LastName,
		t.Number,
		t.Balance,
		t.CreateAt,
	)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", res)
	return nil
}

func (s PostgresStorage) Delete(i int) error {
	query := `delete FROM accounts WHERE id=$1`
	_, err := s.db.Exec(query, i)
	if err != nil {
		return err
	}

	return nil
}

func (s PostgresStorage) Update(t *Account) error {
	//TODO implement me
	panic("implement me")
}

func (s PostgresStorage) GetById(i int) (*Account, error) {
	query := `select * from accounts where id=$1`
	res := new(Account)
	err := s.db.QueryRow(query, i).Scan(&res.ID, &res.FirstName, &res.LastName, &res.Number, &res.Balance, &res.Transactions, &res.CreateAt)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v\n", res)
	return res, nil
}

func (s PostgresStorage) GetAccountWithTransactions(accountID int) (*Account, error) {
	account := &Account{}
	query := `SELECT * FROM accounts WHERE id = $1`
	err := s.db.QueryRow(query, accountID).Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreateAt)
	if err != nil {
		return nil, err
	}

	query = `SELECT * FROM transactions WHERE account_id = $1`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	for rows.Next() {
		transaction := Transaction{}
		err := rows.Scan(&transaction.ID, &transaction.AccountId, &transaction.RecipientId, &transaction.Amount, &transaction.TransactionDate)
		if err != nil {
			return nil, err
		}
		account.Transactions = append(account.Transactions, transaction)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *PostgresStorage) GetTransactionWithDetails(transactionID int) (*Transaction, error) {
	transaction := &Transaction{}
	query := `SELECT * FROM transactions WHERE id = $1`
	err := s.db.QueryRow(query, transactionID).Scan(&transaction.ID, &transaction.AccountId, &transaction.RecipientId, &transaction.Amount, &transaction.TransactionDate)
	if err != nil {
		return nil, err
	}

	query = `SELECT * FROM transaction_details WHERE transaction_id = $1`
	rows, err := s.db.Query(query, transactionID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		detail := TransactionDetails{}
		var tagString string
		err := rows.Scan(&detail.ID, &detail.TransactionId, &detail.Description, &tagString)
		if err != nil {
			return nil, err
		}
		if tagString == "" {
			return nil, err
		}

		tags := strings.Split(tagString, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		detail.Tags = tags
		transaction.Details = append(transaction.Details, detail)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return transaction, nil
}

func (s *PostgresStorage) CreateTransaction(trans *Transaction) error {
	query := `insert into transactions (account_id, recipient_id, amount, transaction_date) values ($1, $2,$3, $4)`
	_, err := s.db.Exec(query, trans.AccountId, trans.RecipientId, trans.Amount, trans.TransactionDate)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) CreateTransactionDetails(details *TransactionDetails) error {
	query := `insert into transaction_details (transaction_id, description, tags) values ($1, $2, $3)`
	_, err := s.db.Exec(query, details.TransactionId, details.Description, strings.Join(details.Tags, ","))
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) DoTransfer(req MakeTransactionRequest) error {
	tx, err := s.db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// make sure source account exists
	if s.DoesAccountExists(req.AccountId) {
		// update source account
		_, err = tx.Exec("UPDATE accounts set balance = balance - $1 where id = $2", req.Amount, req.AccountId)
		if err != nil {
			_ = tx.Rollback()
			log.Fatal(err)
		}
	}

	// make sure recipient account exists
	if s.DoesAccountExists(req.RecipientId) {
		// update recipient account
		_, err = tx.Exec("UPDATE accounts SET balance = balance + $1 WHERE id = $2", req.Amount, req.RecipientId)
		if err != nil {
			_ = tx.Rollback()
			log.Fatal(err)
		}
	}

	// create transaction for source account
	err = s.CreateTransaction(&Transaction{
		AccountId:       req.AccountId,
		RecipientId:     req.RecipientId,
		Amount:          req.Amount,
		TransactionDate: time.Now().UTC(),
	})
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = s.CreateTransaction(&Transaction{
		AccountId:       req.RecipientId,
		RecipientId:     req.RecipientId,
		Amount:          req.Amount,
		TransactionDate: time.Now().UTC(),
	})

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func (s *PostgresStorage) UpdateTransactionDetails(id int, req *TransactionDetailsRequest) error {
	details := new(TransactionDetails)
	getTransactionQuery := `select * from transaction_details where transaction_id = $1`
	res, err := s.db.Query(getTransactionQuery, id)
	if err != nil {
		return err
	}

	for res.Next() {
		var tagString string
		err = res.Scan(&details.ID, &details.TransactionId, &details.Description, &tagString)
		if err != nil {
			return err
		}
		details.Tags = strings.Split(tagString, ",")
	}

	if req.Description != "" {
		details.Description = req.Description
	} else {
		req.Description = details.Description
	}

	if len(req.Tags) != 0 {
		for _, tag := range req.Tags {
			details.Tags = append(details.Tags, tag)
		}
	} else {
		for _, tag := range details.Tags {
			req.Tags = append(req.Tags, tag)
		}
	}

	query := `update transaction_details
		set description = $1,
			tags        = $2
		where transaction_id = $3
	`
	_, err = s.db.Exec(query, details.Description, strings.Join(details.Tags, ","), id)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) GetAccountByNumber(number int) (*Account, error) {
	query := `select * from accounts where number = $1`
	rows, err := s.db.Query(query, number)
	if err != nil {
		return nil, err
	}
	account := new(Account)
	for rows.Next() {
		err := rows.Scan(&account.ID, &account.FirstName, &account.LastName, &account.Number, &account.Balance, &account.CreateAt)
		if err != nil {
			return nil, err
		}
	}

	return account, nil
}

func (s *PostgresStorage) DoesTransactionExists(id int) (bool, error) {
	var exists bool
	query := `select exists(select 1 from transactions where id = $1)`
	err := s.db.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (s *PostgresStorage) DoesAccountExists(id int) bool {
	var exists bool
	query := `select exists(select 1 from accounts where id = $1)`
	err := s.db.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false
	}

	return exists
}

func NewPostgresStorage() (*PostgresStorage, error) {
	connStr := "user=postgres dbname=postgres password=gobank sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStorage{
		db: db,
	}, nil
}
